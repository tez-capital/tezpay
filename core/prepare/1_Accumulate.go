package prepare

import (
	"github.com/echa/log"
	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/core/estimate"
	"github.com/tez-capital/tezpay/utils"
)

func AccumulatePayouts(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	if ctx.PayoutBlueprints == nil {
		return nil, constants.ErrMissingPayoutBlueprint
	}
	if !options.Accumulate {
		return ctx, nil
	}

	payouts := make([]common.PayoutRecipe, 0, len(ctx.StageData.ValidPayouts))
	accumulatedPayouts := make([]common.PayoutRecipe, 0, len(ctx.StageData.ValidPayouts))
	grouped := lo.GroupBy(ctx.StageData.ValidPayouts, func(payout common.PayoutRecipe) string {
		return payout.GetIdentifier()
	})

	for k, groupedPayouts := range grouped {
		if k == "" || len(groupedPayouts) <= 1 {
			payouts = append(payouts, groupedPayouts...)
			continue
		}

		basePayout := groupedPayouts[0]
		groupedPayouts = groupedPayouts[1:]
		for _, payout := range groupedPayouts {
			combined, err := basePayout.Combine(&payout)
			if err != nil {
				return nil, err
			}
			accumulatedPayouts = append(accumulatedPayouts, payout)
			basePayout = *combined
		}

		payouts = append(payouts, basePayout) // add the combined
	}

	payoutKey := ctx.GetSigner().GetKey()

	estimateContext := &estimate.EstimationContext{
		PayoutKey:     payoutKey,
		Collector:     ctx.GetCollector(),
		Configuration: ctx.configuration,
		BatchMetadataDeserializationGasLimit: lo.Max(lo.Map(ctx.PayoutBlueprints, func(blueprint *common.CyclePayoutBlueprint, _ int) int64 {
			return blueprint.BatchMetadataDeserializationGasLimit
		})),
	}

	// get new estimates
	payouts = lo.Map(estimate.EstimateTransactionFees(utils.MapToPointers(payouts), estimateContext), func(result estimate.EstimateResult[*common.PayoutRecipe], _ int) common.PayoutRecipe {
		if result.Error != nil {
			log.Warnf("failed to estimate tx costs to '%s' (delegator: '%s', amount %d, kind '%s')\nerror: %s", result.Transaction.Recipient, payoutKey.Address(), result.Transaction.Amount.Int64(), result.Transaction.TxKind, result.Error.Error())
			result.Transaction.IsValid = false
			result.Transaction.Note = string(enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS)
		}

		candidate := result.Transaction
		if candidate.TxKind == enums.PAYOUT_TX_KIND_TEZ {
			if !candidate.TxFeeCollected {
				candidate.Amount = candidate.Amount.Add64(candidate.OpLimits.GetOperationFeesWithoutAllocation() - result.Result.GetOperationFeesWithoutAllocation())
			}
			if !candidate.AllocationFeeCollected {
				candidate.Amount = candidate.Amount.Add64(candidate.OpLimits.GetAllocationFee() - result.Result.GetAllocationFee())
			}
		}

		result.Transaction.OpLimits = result.Result
		return *result.Transaction
	})

	ctx.StageData.ValidPayouts = payouts
	ctx.StageData.AccumulatedPayouts = accumulatedPayouts

	return ctx, nil
}
