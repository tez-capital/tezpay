package generate

import (
	"errors"
	"fmt"
	"time"

	"log/slog"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/extension"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/tezos"
)

type CheckBalanceHookData struct {
	SkipTezCheck bool                                  `json:"skip_tez_check"`
	IsSufficient bool                                  `json:"is_sufficient"`
	Message      string                                `json:"message"`
	Payouts      []PayoutCandidateWithBondAmountAndFee `json:"payouts"`
}

func checkBalanceWithHook(data *CheckBalanceHookData) error {
	err := extension.ExecuteHook(enums.EXTENSION_HOOK_CHECK_BALANCE, "0.1", data)
	if err != nil {
		return err
	}
	return nil
}

func checkBalanceWithCollector(data *CheckBalanceHookData, ctx *PayoutGenerationContext) error {
	if data.SkipTezCheck { // skip tez check for cases when pervious hook already checked it
		return nil
	}
	payableBalance, err := ctx.GetCollector().GetBalance(ctx.PayoutKey.Address())
	if err != nil {
		return err
	}

	configuration := ctx.GetConfiguration()

	totalPayouts := len(lo.Filter(data.Payouts, func(candidate PayoutCandidateWithBondAmountAndFee, _ int) bool {
		return !candidate.IsInvalid
	}))

	// calculate bonds and fees portion
	bondsPortionToBeForwarded := lo.Sum(lo.Values(configuration.IncomeRecipients.Bonds))
	feesPortionToBeForwarded := lo.Sum(lo.Values(configuration.IncomeRecipients.Fees))

	// add all bonds, fees and donations destinations
	totalPayouts = totalPayouts + len(configuration.IncomeRecipients.Bonds) + len(configuration.IncomeRecipients.Fees) + utils.Max(len(configuration.IncomeRecipients.Donations), 1)

	requiredbalance := lo.Reduce(data.Payouts, func(agg tezos.Z, candidate PayoutCandidateWithBondAmountAndFee, _ int) tezos.Z {
		if candidate.TxKind == enums.PAYOUT_TX_KIND_TEZ {
			return agg.Add(candidate.BondsAmount)
		}
		return agg
	}, tezos.Zero)
	// bonds * bondsPortionToBeForwarded
	bondsToBeForwarded := ctx.StageData.BakerBondsAmount.Mul64(int64(bondsPortionToBeForwarded * 1000000)).Div64(1000000)
	// fees * feesPortionToBeForwarded
	feesToBeForwarded := ctx.StageData.BakerFeesAmount.Mul64(int64(feesPortionToBeForwarded * 1000000)).Div64(1000000)

	// add bonds,fees and donations to required balance
	requiredbalance = requiredbalance.Add(bondsToBeForwarded).Add(feesToBeForwarded).Add(ctx.StageData.DonateBondsAmount)
	requiredbalance = requiredbalance.Add(tezos.NewZ(constants.PAYOUT_FEE_BUFFER).Mul64(int64(totalPayouts)))

	diff := payableBalance.Sub(requiredbalance)
	if diff.IsNeg() || diff.IsZero() {
		data.IsSufficient = false
		data.Message = fmt.Sprintf("required: %s, available: %s", requiredbalance, payableBalance)
	}
	return nil
}

func runBalanceCheck(ctx *PayoutGenerationContext, logger *slog.Logger, check func(*CheckBalanceHookData) error, data *CheckBalanceHookData, options *common.GeneratePayoutsOptions) error {
	notificatorTrigger := 0
	for {
		// we reset values before each check so we get relevant data for this check only
		data.IsSufficient = true
		data.Message = ""

		if err := check(data); err != nil {
			if options.WaitForSufficientBalance {
				logger.Error("failed to check balance, retrying in 5 minutes", "error", err.Error())
				time.Sleep(time.Minute * 5)
				continue
			}
			return err
		}

		if !data.IsSufficient {
			if options.WaitForSufficientBalance {
				logger.Warn("insufficient balance, retrying in 5 minutes...", "message", data.Message)
				if notificatorTrigger%12 == 0 { // every hour
					ctx.AdminNotify(fmt.Sprintf("insufficient balance - %s", data.Message))
				}
				time.Sleep(time.Minute * 5)
				notificatorTrigger++
				continue
			}
			return errors.Join(constants.ErrInsufficientBalance, errors.New(data.Message))
		}
		break
	}
	return nil
}

/*
Technically we could calculate real required balance by checking all payouts and fees and donations in final stage
but because of potential changes of transaction fees (on-chain state changes) it would not be accurate anyway.
So we just try to estimate with a buffer which should be enough for most cases.
*/

func CheckSufficientBalance(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	logger := ctx.logger.With("phase", "check_sufficient_balance")
	if options.SkipBalanceCheck { // skip
		return ctx, nil
	}

	logger.Debug("checking sufficient balance")
	hookResponse := CheckBalanceHookData{
		IsSufficient: true,
		Payouts:      ctx.StageData.PayoutCandidatesWithBondAmountAndFees,
	}

	checks := []func(*CheckBalanceHookData) error{
		func(data *CheckBalanceHookData) error {
			logger.Debug("checking balance with hook")
			return checkBalanceWithHook(data)
		},
		func(data *CheckBalanceHookData) error {
			logger.Debug("checking tez balance with collector")
			return checkBalanceWithCollector(data, ctx)
		},
	}

	for _, check := range checks {
		err := runBalanceCheck(ctx, logger, check, &hookResponse, options)
		if err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}
