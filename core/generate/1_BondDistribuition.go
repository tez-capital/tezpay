package generate

import (
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/extension"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/tezos"

	"github.com/samber/lo"
)

type AfterBondsDistributedHookData struct {
	Cycle      int64                           `json:"cycle"`
	Candidates []PayoutCandidateWithBondAmount `json:"candidates"`
}

func ExecuteAfterBondsDistributed(data *AfterBondsDistributedHookData) error {
	return extension.ExecuteHook(enums.EXTENSION_HOOK_AFTER_BONDS_DISTRIBUTED, "0.2", data)
}

func getBakerBondsAmount(cycleData *common.BakersCycleData, effectiveDelegatorsDelegatedBalance tezos.Z, configuration *configuration.RuntimeConfiguration) tezos.Z {
	bakerDelegatedBalance := cycleData.GetBakerDelegatedBalance()
	totalRewards := cycleData.GetTotalDelegatedRewards(configuration.PayoutConfiguration.PayoutMode)

	totalDelegatedBalance := effectiveDelegatorsDelegatedBalance.Add(bakerDelegatedBalance)

	maximumDelegated := cycleData.GetBakerStakedBalance().Mul64(constants.DELEGATION_CAPACITY_FACTOR)
	if maximumDelegated.Sub(totalDelegatedBalance).IsNeg() && configuration.Overdelegation.IsProtectionEnabled { // overdelegated and protection enabled
		totalDelegatedBalance = maximumDelegated // this will bracket the totalDelegatedBalance to maximumDelegated and baker takes his full share from delegated balance, the rest is dilluted
	}
	if maximumDelegated.IsLess(bakerDelegatedBalance) && configuration.Overdelegation.IsProtectionEnabled {
		// this is just to give sane results in case bakers is overdelegated to itself
		// without this tezpay reports negative rewards for delegators in such cases
		bakerDelegatedBalance = maximumDelegated
	}
	bakerDelegatedBondsAmount := totalRewards.Mul(bakerDelegatedBalance).Div(totalDelegatedBalance)
	return bakerDelegatedBondsAmount
}

func isDelegatorEligibleForBonds(candidate PayoutCandidate, configuration *configuration.RuntimeConfiguration) bool {
	if candidate.IsInvalid {
		if candidate.InvalidBecause == enums.INVALID_DELEGATOR_IGNORED {
			return false
		}
		if configuration.Delegators.Requirements.BellowMinimumBalanceRewardDestination == enums.REWARD_DESTINATION_EVERYONE && candidate.InvalidBecause == enums.INVALID_DELEGATOR_LOW_BAlANCE {
			return false
		}
	}
	return true
}

func DistributeBonds(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	configuration := ctx.GetConfiguration()
	logger := ctx.logger.With("phase", "distribute_bonds")

	logger.Debug("distributing bonds")

	candidates := ctx.StageData.PayoutCandidates
	totalDelegatorsDelegatedBalance := lo.Reduce(candidates, func(total tezos.Z, candidate PayoutCandidate, _ int) tezos.Z {
		// of all delegators, including invalids, except ignored and possibly excluding bellow minimum balance
		if !isDelegatorEligibleForBonds(candidate, configuration) {
			return total
		}
		return total.Add(candidate.GetDelegatedBalance())
	}, tezos.NewZ(0))

	bakerBonds := getBakerBondsAmount(ctx.StageData.CycleData, totalDelegatorsDelegatedBalance, configuration)
	availableRewards := ctx.StageData.CycleData.GetTotalDelegatedRewards(configuration.PayoutConfiguration.PayoutMode).Sub(bakerBonds)

	ctx.StageData.PayoutCandidatesWithBondAmount = lo.Map(candidates, func(candidate PayoutCandidate, _ int) PayoutCandidateWithBondAmount {
		if !isDelegatorEligibleForBonds(candidate, configuration) {
			return PayoutCandidateWithBondAmount{
				PayoutCandidate: candidate,
				BondsAmount:     tezos.Zero,
			}
		}

		delegatorBondsAmount := availableRewards.Mul(candidate.GetDelegatedBalance()).Div(totalDelegatorsDelegatedBalance)
		utils.AssertZAmountPositiveOrZero(delegatorBondsAmount)

		return PayoutCandidateWithBondAmount{
			PayoutCandidate: candidate,
			BondsAmount:     delegatorBondsAmount,
			TxKind:          enums.PAYOUT_TX_KIND_TEZ,
		}
	})

	bondsDonate := utils.GetZPortion(bakerBonds, configuration.IncomeRecipients.DonateBonds)
	ctx.StageData.BakerBondsAmount = bakerBonds.Sub(bondsDonate)
	ctx.StageData.DonateBondsAmount = bondsDonate
	utils.AssertZAmountPositiveOrZero(ctx.StageData.BakerBondsAmount)
	utils.AssertZAmountPositiveOrZero(ctx.StageData.DonateBondsAmount)

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
