package stages

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"

	"github.com/samber/lo"
)

func getBakerBondsAmount(cycleData *common.BakersCycleData, overdelegationProtection bool) tezos.Z {
	bakerBalance := cycleData.GetBakerBalance()
	totalRewards := cycleData.GetTotalRewards()

	overdelegationLimit := cycleData.FrozenDeposit
	if overdelegationLimit.IsZero() {
		overdelegationLimit = bakerBalance
	}
	bakerAmount := totalRewards.Div64(constants.DELEGATION_CAPACITY_FACTOR)
	if overdelegationLimit.Sub(cycleData.StakingBalance).IsNeg() || !overdelegationProtection { // not overdelegated
		bakerAmount = totalRewards.Mul(bakerBalance).Div(cycleData.StakingBalance)
	}
	return bakerAmount
}

func distributeBonds(ctx Context) (Context, error) {
	configuration := ctx.GetConfiguration()
	log.Debugf("distributing bonds")

	bakerBonds := getBakerBondsAmount(ctx.CycleData, configuration.Overdelegation.IsProtectionEnabled)

	candidates := ctx.StageData.PayoutCandidates

	availableRewards := ctx.CycleData.GetTotalRewards().Sub(bakerBonds)
	totalBalance := lo.Reduce(candidates, func(total tezos.Z, candidate PayoutCandidate, _ int) tezos.Z { // of all delegators, including invalids
		return total.Add(candidate.Balance)
	}, tezos.NewZ(0))

	ctx.StageData.PayoutCandidatesWithBondAmount = lo.Map(candidates, func(candidate PayoutCandidate, _ int) PayoutCandidateWithBondAmount {
		if candidate.IsInvalid {
			return PayoutCandidateWithBondAmount{
				PayoutCandidate: candidate,
				BondsAmount:     tezos.Zero,
			}
		}
		return PayoutCandidateWithBondAmount{
			PayoutCandidate: candidate,
			BondsAmount:     availableRewards.Mul(candidate.Balance).Div(totalBalance),
		}
	})

	bondsDonate := utils.GetZPortion(bakerBonds, configuration.IncomeRecipients.Donate)
	ctx.StageData.BakerBondsAmount = bakerBonds.Sub(bondsDonate)
	ctx.StageData.DonateBondsAmount = bondsDonate

	return ctx, nil
}

var DistributeBonds = WrapStage(distributeBonds)
