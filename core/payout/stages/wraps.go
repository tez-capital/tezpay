package stages

import (
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
	"github.com/samber/lo"
)

type Stage func(ctx Context) (Context, error)

type WrappedStageResult struct {
	Ctx Context
	Err error
}
type WrappedStage func(previous WrappedStageResult) WrappedStageResult

func (result WrappedStageResult) ExecuteStage(stage WrappedStage) WrappedStageResult {
	return stage(result)
}

func (result WrappedStageResult) ExecuteStages(stages ...WrappedStage) WrappedStageResult {
	for _, stage := range stages {
		result = stage(result)
	}
	return result
}

func WrapContextWithError(ctx Context, err error) WrappedStageResult {
	return WrappedStageResult{
		Ctx: ctx,
		Err: err,
	}
}

func WrapStage(stage Stage) WrappedStage {
	return func(previous WrappedStageResult) WrappedStageResult {
		if previous.Err != nil {
			return previous
		}
		ctx, err := stage(previous.Ctx)
		return WrapContextWithError(ctx, err)
	}
}

func (result WrappedStageResult) Unwrap() (Context, error) {
	return result.Ctx, result.Err
}

func sumValidPayoutsAmount(payouts []common.PayoutRecipe) tezos.Z {
	return lo.Reduce(payouts, func(agg tezos.Z, payout common.PayoutRecipe, _ int) tezos.Z {
		if !payout.IsValid {
			return agg
		}
		return agg.Add(payout.Amount)
	}, tezos.Zero)
}

func (result WrappedStageResult) ToCyclePayoutBlueprint() (*common.CyclePayoutBlueprint, error) {
	if result.Err != nil {
		return nil, result.Err
	}

	return &common.CyclePayoutBlueprint{
		Cycle:   result.Ctx.Cycle,
		Payouts: result.Ctx.StageData.Payouts,
		Summary: common.CyclePayoutSummary{
			Cycle:              result.Ctx.Cycle,
			Delegators:         len(result.Ctx.CycleData.Delegators),
			StakingBalance:     result.Ctx.CycleData.StakingBalance,
			EarnedFees:         result.Ctx.CycleData.BlockFees,
			EarnedRewards:      result.Ctx.CycleData.BlockRewards.Add(result.Ctx.CycleData.EndorsementRewards),
			DistributedRewards: sumValidPayoutsAmount(result.Ctx.StageData.Payouts),
			BondIncome:         result.Ctx.StageData.BakerBondsAmount,
			FeeIncome:          result.Ctx.StageData.BakerFeesAmount,
			IncomeTotal:        result.Ctx.StageData.BakerBondsAmount.Add(result.Ctx.StageData.BakerFeesAmount),
			DonatedBonds:       result.Ctx.StageData.DonateBondsAmount,
			DonatedFees:        result.Ctx.StageData.DonateFeesAmount,
			DonatedTotal:       result.Ctx.StageData.DonateFeesAmount.Add(result.Ctx.StageData.DonateBondsAmount),
			Timestamp:          time.Now(),
		},
	}, nil
}
