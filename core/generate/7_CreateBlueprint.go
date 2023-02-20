package generate

import (
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/samber/lo"
)

func sumValidPayoutsAmount(payouts []common.PayoutRecipe) tezos.Z {
	return lo.Reduce(payouts, func(agg tezos.Z, payout common.PayoutRecipe, _ int) tezos.Z {
		if !payout.IsValid {
			return agg
		}
		return agg.Add(payout.Amount)
	}, tezos.Zero)
}

func CreateBlueprint(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (result *PayoutGenerationContext, err error) {
	stageData := ctx.StageData

	stageData.PayoutBlueprint = &common.CyclePayoutBlueprint{
		Cycle:   options.Cycle,
		Payouts: stageData.Payouts,
		Summary: common.CyclePayoutSummary{
			Cycle:              options.Cycle,
			Delegators:         len(stageData.CycleData.Delegators),
			PaidDelegators:     stageData.PaidDelegators,
			StakingBalance:     stageData.CycleData.StakingBalance,
			EarnedFees:         stageData.CycleData.BlockFees,
			EarnedRewards:      stageData.CycleData.BlockRewards.Add(stageData.CycleData.EndorsementRewards),
			DistributedRewards: sumValidPayoutsAmount(stageData.Payouts),
			BondIncome:         stageData.BakerBondsAmount,
			FeeIncome:          stageData.BakerFeesAmount,
			IncomeTotal:        stageData.BakerBondsAmount.Add(stageData.BakerFeesAmount),
			DonatedBonds:       stageData.DonateBondsAmount,
			DonatedFees:        stageData.DonateFeesAmount,
			DonatedTotal:       stageData.DonateFeesAmount.Add(stageData.DonateBondsAmount),
			Timestamp:          time.Now(),
		},
	}
	return ctx, nil
}
