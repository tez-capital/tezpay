package generate

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/extension"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"

	"github.com/samber/lo"
)

type AfterBondsDistributedHookData struct {
	Cycle      int64                           `json:"cycle"`
	Candidates []PayoutCandidateWithBondAmount `json:"candidates"`
}

func ExecuteAfterBondsDistributed(data *AfterBondsDistributedHookData) error {
	return extension.ExecuteHook(enums.EXTENSION_HOOK_AFTER_BONDS_DISTRIBUTED, "0.2", data)
}

func getBakerBondsAmount(cycleData *common.BakersCycleData, effectiveDelegatorsStakingBalance tezos.Z, configuration *configuration.RuntimeConfiguration) tezos.Z {
	bakerBalance := cycleData.GetBakerBalance()
	totalRewards := cycleData.GetTotalRewards(configuration.PayoutConfiguration.PayoutMode)

	overdelegationLimit := cycleData.FrozenDepositLimit
	if overdelegationLimit.IsZero() {
		overdelegationLimit = bakerBalance
	}
	bakerAmount := totalRewards.Div64(constants.DELEGATION_CAPACITY_FACTOR)
	stakingBalance := effectiveDelegatorsStakingBalance.Add(bakerBalance)
	if !overdelegationLimit.Mul64(constants.DELEGATION_CAPACITY_FACTOR).Sub(stakingBalance).IsNeg() || !configuration.Overdelegation.IsProtectionEnabled { // not overdelegated
		bakerAmount = totalRewards.Mul(bakerBalance).Div(stakingBalance)
	}
	return bakerAmount
}

func DistributeBonds(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	configuration := ctx.GetConfiguration()
	log.Debugf("distributing bonds")

	candidates := ctx.StageData.PayoutCandidates
	effectiveStakingBalance := lo.Reduce(candidates, func(total tezos.Z, candidate PayoutCandidate, _ int) tezos.Z {
		// of all delegators, including invalids, except ignored
		if candidate.IsInvalid && candidate.InvalidBecause == enums.INVALID_DELEGATOR_IGNORED {
			return total
		}
		return total.Add(candidate.Balance)
	}, tezos.NewZ(0))

	bakerBonds := getBakerBondsAmount(ctx.StageData.CycleData, effectiveStakingBalance, configuration)
	availableRewards := ctx.StageData.CycleData.GetTotalRewards(configuration.PayoutConfiguration.PayoutMode).Sub(bakerBonds)

	ctx.StageData.PayoutCandidatesWithBondAmount = lo.Map(candidates, func(candidate PayoutCandidate, _ int) PayoutCandidateWithBondAmount {
		if candidate.IsInvalid {
			return PayoutCandidateWithBondAmount{
				PayoutCandidate: candidate,
				BondsAmount:     tezos.Zero,
			}
		}
		return PayoutCandidateWithBondAmount{
			PayoutCandidate: candidate,
			BondsAmount:     availableRewards.Mul(candidate.Balance).Div(effectiveStakingBalance),
			TxKind:          enums.PAYOUT_TX_KIND_TEZ,
		}
	})

	bondsDonate := utils.GetZPortion(bakerBonds, configuration.IncomeRecipients.Donate)
	ctx.StageData.BakerBondsAmount = bakerBonds.Sub(bondsDonate)
	ctx.StageData.DonateBondsAmount = bondsDonate

	hookData := &AfterBondsDistributedHookData{
		Cycle:      options.Cycle,
		Candidates: ctx.StageData.PayoutCandidatesWithBondAmount,
	}
	err := ExecuteAfterBondsDistributed(hookData)
	if err != nil {
		return ctx, err
	}
	ctx.StageData.PayoutCandidatesWithBondAmount = hookData.Candidates

	return ctx, nil
}
