package generate

import (
	"fmt"
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

/*
Technically we could calculate real required balance by checking all payouts and fees and donations in final stage
but because of potential changes of transaction fees (on-chain state changes) it would not be accurate anyway.
So we just try to estimate with a buffer which should be enough for most cases.
*/

func CheckSufficientBalance(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	configuration := ctx.GetConfiguration()
	if options.SkipBalanceCheck { // skip
		return ctx, nil
	}

	log.Debugf("checking for sufficient balance")
	candidates := ctx.StageData.PayoutCandidatesWithBondAmount

	totalPayouts := len(lo.Filter(candidates, func(candidate PayoutCandidateWithBondAmount, _ int) bool {
		return !candidate.IsInvalid
	}))
	// add all bonds, fees and donations destinations
	totalPayouts = totalPayouts + len(configuration.IncomeRecipients.Bonds) + len(configuration.IncomeRecipients.Fees) + utils.Max(len(configuration.IncomeRecipients.Donations), 1)

	requiredbalance := lo.Reduce(candidates, func(agg tezos.Z, candidate PayoutCandidateWithBondAmount, _ int) tezos.Z {
		return agg.Add(candidate.BondsAmount)
	}, tezos.Zero)

	requiredbalance = ctx.StageData.BakerBondsAmount.Add(requiredbalance)
	requiredbalance = requiredbalance.Add(tezos.NewZ(constants.PAYOUT_FEE_BUFFER).Mul64(int64(totalPayouts)))

	checked := false
	notificatorTrigger := 0
	for !checked || options.WaitForSufficientBalance {

		payableBalance, err := ctx.GetCollector().GetBalance(ctx.PayoutKey.Address())
		if err != nil {
			if options.WaitForSufficientBalance {
				log.Errorf("failed to check balance - %s, waiting 5 minutes...", err.Error())
				time.Sleep(time.Minute * 5)
				continue
			}
			return ctx, err
		}

		diff := payableBalance.Sub(requiredbalance)
		if diff.IsNeg() || diff.IsZero() { // zero is probably too on edge so better to keep checking for zero
			if options.WaitForSufficientBalance {
				log.Warnf("insufficient balance - needs at least %s but only has %s, waiting 5 minutes...", utils.MutezToTezS(requiredbalance.Int64()), utils.MutezToTezS(payableBalance.Int64()))
				if notificatorTrigger%12 == 0 { // every hour
					ctx.AdminNotify(fmt.Sprintf("insufficient balance - needs at least %s but only has %s", utils.MutezToTezS(requiredbalance.Int64()), utils.MutezToTezS(payableBalance.Int64())))
				}
				time.Sleep(time.Minute * 5)
				notificatorTrigger++
				continue
			}
			return ctx, fmt.Errorf("insufficient balance - needs at least %s but only has %s", utils.MutezToTezS(requiredbalance.Int64()), utils.MutezToTezS(payableBalance.Int64()))
		}
		break
	}

	return ctx, nil
}
