package prepare

import (
	"slices"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/extension"
)

type AfterPayoutsAccumulated struct {
	Recipes                       []common.PayoutRecipe             `json:"recipes"`
	AccumulatedValidPayouts       []*common.AccumulatedPayoutRecipe `json:"payouts"`
	InvalidPayouts                []common.PayoutRecipe             `json:"invalid_payouts"`
	ReportsOfPastSuccesfulPayouts []common.PayoutReport             `json:"reports_of_past_succesful_payouts"`
}

func ExecuteAfterPayoutsAccumulated(data *AfterPayoutsAccumulated) error {
	return extension.ExecuteHook(enums.EXTENSION_HOOK_AFTER_PAYOUTS_ACCUMULATED, "0.1", data)
}

func AccumulatePayouts(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	if ctx.PayoutBlueprints == nil {
		return nil, constants.ErrMissingPayoutBlueprint
	}
	// copy valid payouts to accumulated in case we return early
	ctx.StageData.AccumulatedValidPayouts = lo.Map(ctx.StageData.ValidPayouts, func(payout common.PayoutRecipe, _ int) *common.AccumulatedPayoutRecipe {
		return payout.AsAccumulated()
	})
	if !options.Accumulate {
		return ctx, nil
	}

	cycles := lo.Reduce(ctx.StageData.ValidPayouts, func(acc []int64, payout common.PayoutRecipe, _ int) []int64 {
		if !slices.Contains(acc, payout.Cycle) {
			acc = append(acc, payout.Cycle)
		}
		return acc
	}, []int64{})
	if len(cycles) <= 1 { // nothing to accumulate
		return ctx, nil
	}

	logger := ctx.logger.With("phase", "accumulate_payouts")
	logger.Info("accumulating payouts")

	payouts := make([]*common.AccumulatedPayoutRecipe, 0, len(ctx.StageData.ValidPayouts))
	grouped := lo.GroupBy(ctx.StageData.ValidPayouts, func(payout common.PayoutRecipe) string {
		return payout.GetIdentifier()
	})

	for k, groupedPayouts := range grouped {
		if k == "" || len(groupedPayouts) <= 1 {
			payouts = append(payouts, lo.Map(groupedPayouts, func(payout common.PayoutRecipe, _ int) *common.AccumulatedPayoutRecipe {
				return payout.AsAccumulated()
			})...)
			continue
		}

		basePayout := groupedPayouts[0].AsAccumulated()
		groupedPayouts = groupedPayouts[1:]
		for _, payout := range groupedPayouts {
			combined, err := basePayout.Add(&payout)
			if err != nil {
				return nil, err
			}
			basePayout = combined
		}

		payouts = append(payouts, basePayout) // add the combined
	}

	hookData := &AfterPayoutsAccumulated{
		Recipes: lo.Reduce(ctx.PayoutBlueprints, func(agg []common.PayoutRecipe, blueprint *common.CyclePayoutBlueprint, _ int) []common.PayoutRecipe {
			return append(agg, blueprint.Payouts...)
		}, make([]common.PayoutRecipe, 0)),
		AccumulatedValidPayouts:       payouts,
		InvalidPayouts:                ctx.StageData.InvalidPayouts,
		ReportsOfPastSuccesfulPayouts: ctx.StageData.ReportsOfPastSuccesfulPayouts,
	}
	err := ExecuteAfterPayoutsAccumulated(hookData)
	if err != nil {
		return ctx, err
	}

	ctx.StageData.AccumulatedValidPayouts = hookData.AccumulatedValidPayouts
	ctx.StageData.InvalidPayouts = hookData.InvalidPayouts
	ctx.StageData.ReportsOfPastSuccesfulPayouts = hookData.ReportsOfPastSuccesfulPayouts

	// payoutKey := ctx.GetSigner().GetKey()

	// estimateContext := &estimate.EstimationContext{
	// 	PayoutKey:     payoutKey,
	// 	Collector:     ctx.GetCollector(),
	// 	Configuration: ctx.configuration,
	// 	BatchMetadataDeserializationGasLimit: lo.Max(lo.Map(ctx.PayoutBlueprints, func(blueprint *common.CyclePayoutBlueprint, _ int) int64 {
	// 		return blueprint.BatchMetadataDeserializationGasLimit
	// 	})),
	// }

	// // get new estimates
	// payouts = lo.Map(estimate.EstimateTransactionFees(utils.MapToPointers(payouts), estimateContext), func(result estimate.EstimateResult[*common.PayoutRecipe], _ int) common.PayoutRecipe {
	// 	if result.Error != nil {
	// 		slog.Warn("failed to estimate tx costs", "recipient", result.Transaction.Recipient, "delegator", payoutKey.Address(), "amount", result.Transaction.Amount.Int64(), "kind", result.Transaction.TxKind, "error", result.Error)
	// 		result.Transaction.IsValid = false
	// 		result.Transaction.Note = string(enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS)
	// 	}

	// 	candidate := result.Transaction
	// 	if candidate.TxKind == enums.PAYOUT_TX_KIND_TEZ {
	// 		if !candidate.TxFeeCollected {
	// 			candidate.Amount = candidate.Amount.Add64(candidate.OpLimits.GetOperationFeesWithoutAllocation() - result.OpLimits.GetOperationFeesWithoutAllocation())
	// 		}
	// 		if !candidate.AllocationFeeCollected {
	// 			candidate.Amount = candidate.Amount.Add64(candidate.OpLimits.GetAllocationFee() - result.OpLimits.GetAllocationFee())
	// 		}
	// 	}

	// 	result.Transaction.OpLimits = result.OpLimits
	// 	return *result.Transaction
	// })

	// ctx.StageData.ValidPayouts = payouts

	// ctx.StageData.ValidPayouts, ctx.StageData.InvalidPayouts, ctx.StageData.ReportsOfPastSuccesfulPayouts = hookData.ValidPayouts, hookData.InvalidPayouts, hookData.ReportsOfPastSuccesfulPayouts

	return ctx, nil
}
