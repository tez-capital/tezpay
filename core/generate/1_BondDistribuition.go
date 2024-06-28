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
	bakerDelegatedBondsAmount := totalRewards.Mul(bakerDelegatedBalance).Div(totalDelegatedBalance)

	return bakerDelegatedBondsAmount
}

func DistributeBonds(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	configuration := ctx.GetConfiguration()
	logger := ctx.logger.With("phase", "distribute_bonds")

	logger.Debug("distributing bonds")

	candidates := ctx.StageData.PayoutCandidates
	totalDelegatorsDelegatedBalance := lo.Reduce(candidates, func(total tezos.Z, candidate PayoutCandidate, _ int) tezos.Z {
		// of all delegators, including invalids, except ignored and possibly excluding bellow minimum balance
		if candidate.IsInvalid {
			if candidate.InvalidBecause == enums.INVALID_DELEGATOR_IGNORED {
				return total
			}
			if ctx.configuration.Delegators.Requirements.BellowMinimumBalanceRewardDestination == enums.REWARD_DESTINATION_EVERYONE && candidate.InvalidBecause == enums.INVALID_DELEGATOR_LOW_BAlANCE {
				return total
			}
		}
		return total.Add(candidate.GetDelegatedBalance())
	}, tezos.NewZ(0))

	bakerBonds := getBakerBondsAmount(ctx.StageData.CycleData, totalDelegatorsDelegatedBalance, configuration)
	availableRewards := ctx.StageData.CycleData.GetTotalDelegatedRewards(configuration.PayoutConfiguration.PayoutMode).Sub(bakerBonds)

	ctx.StageData.PayoutCandidatesWithBondAmount = lo.Map(candidates, func(candidate PayoutCandidate, _ int) PayoutCandidateWithBondAmount {
		if candidate.IsInvalid {
			return PayoutCandidateWithBondAmount{
				PayoutCandidate: candidate,
				BondsAmount:     tezos.Zero,
			}
		}
		return PayoutCandidateWithBondAmount{
			PayoutCandidate: candidate,
			BondsAmount:     availableRewards.Mul(candidate.GetDelegatedBalance()).Div(totalDelegatorsDelegatedBalance),
			TxKind:          enums.PAYOUT_TX_KIND_TEZ,
		}
	})

	bondsDonate := utils.GetZPortion(bakerBonds, configuration.IncomeRecipients.DonateBonds)
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
