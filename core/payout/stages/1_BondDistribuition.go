package stages

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core/payout/common"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"

	tezpay_tezos "github.com/alis-is/tezpay/clients/tezos"
	"github.com/samber/lo"
)

func getBakerBondsAmount(cycleData *tezpay_tezos.BakersCycleData, overdelegationProtection bool) tezos.Z {
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

func distributeBonds(ctx common.Context) (common.Context, error) {
	configuration := ctx.GetConfiguration()
	log.Debugf("distributing bonds")

	bakerBonds := getBakerBondsAmount(ctx.CycleData, configuration.Overdelegation.IsProtectionEnabled)

	candidates := ctx.StageData.PayoutCandidates

	availableRewards := ctx.CycleData.GetTotalRewards().Sub(bakerBonds)
	totalBalance := lo.Reduce(candidates, func(total tezos.Z, candidate common.PayoutCandidate, _ int) tezos.Z { // of all delegators, including invalids
		return total.Add(candidate.Balance)
	}, tezos.NewZ(0))

	ctx.StageData.PayoutCandidatesWithBondAmount = lo.Map(candidates, func(candidate common.PayoutCandidate, _ int) common.PayoutCandidateWithBondAmount {
		if candidate.IsInvalid {
			return common.PayoutCandidateWithBondAmount{
				Candidate:   candidate,
				BondsAmount: tezos.Zero,
			}
		}
		return common.PayoutCandidateWithBondAmount{
			Candidate:   candidate,
			BondsAmount: availableRewards.Mul(candidate.Balance).Div(totalBalance),
		}
	})

	bondsDonate := utils.GetZPortion(bakerBonds, configuration.IncomeRecipients.Donate)
	ctx.StageData.BakerBondsAmount = bakerBonds.Sub(bondsDonate)
	ctx.StageData.DonateBondsAmount = bondsDonate

	return ctx, nil
}

var DistributeBonds = common.WrapStage(distributeBonds)
