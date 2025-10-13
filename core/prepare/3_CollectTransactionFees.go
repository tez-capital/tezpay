package prepare

import (
	"log/slog"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/core/estimate"
	"github.com/tez-capital/tezpay/utils"
)

func CollectTransactionFees(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (result *PayoutPrepareContext, err error) {
	logger := ctx.logger.With("phase", "collect_transaction_fees")
	logger.Info("collecting transaction fees")

	estimateContext := &estimate.EstimationContext{
		PayoutKey:                            ctx.PayoutKey,
		Collector:                            ctx.GetCollector(),
		Configuration:                        ctx.configuration,
		BatchMetadataDeserializationGasLimit: ctx.StageData.BatchMetadataDeserializationGasLimit,
	}

	validAccumulatedRecipes := utils.OnlyValidAccumulatedPayouts(ctx.StageData.AccumulatedPayouts)
	// get new estimates
	recipesWithEstimate := lo.Map(estimate.EstimateTransactionFees(validAccumulatedRecipes, estimateContext), func(result estimate.EstimateResult[*common.AccumulatedPayoutRecipe], _ int) *common.AccumulatedPayoutRecipe {
		if result.Error != nil {
			slog.Warn("failed to estimate tx costs", "recipient", result.Transaction.Recipient, "delegator", ctx.PayoutKey.Address(), "amount", result.Transaction.GetAmount().Int64(), "kind", result.Transaction.TxKind, "error", result.Error)
			result.Transaction.IsValid = false
			result.Transaction.Note = string(enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS)
			return result.Transaction
		}

		recipe := result.Transaction
		recipe.OpLimits = result.OpLimits

		if recipe.TxKind == enums.PAYOUT_TX_KIND_TEZ &&
			recipe.Kind == enums.PAYOUT_KIND_DELEGATOR_REWARD { // only delegator rewards are subject to fee collection

			isBakerPayingTxFee := ctx.configuration.PayoutConfiguration.IsPayingTxFee
			isBakerPayingAllocationTxFee := ctx.configuration.PayoutConfiguration.IsPayingAllocationTxFee

			delegatorOverrides := ctx.configuration.Delegators.Overrides
			if delegatorOverride, ok := delegatorOverrides[recipe.Delegator.String()]; ok {
				if delegatorOverride.IsBakerPayingTxFee != nil {
					isBakerPayingTxFee = *delegatorOverride.IsBakerPayingTxFee
				}
				if delegatorOverride.IsBakerPayingAllocationTxFee != nil {
					isBakerPayingAllocationTxFee = *delegatorOverride.IsBakerPayingAllocationTxFee
				}
			}

			bondsAmountBeforeFees := recipe.GetAmount()
			utils.AssertZAmountPositiveOrZero(bondsAmountBeforeFees)

			txFee := result.OpLimits.GetOperationFeesWithoutAllocation()
			allocationFee := result.OpLimits.GetAllocationFee()
			if !isBakerPayingTxFee {
				recipe.SubtractAmount64(txFee)
			}
			if !isBakerPayingAllocationTxFee {
				recipe.SubtractAmount64(allocationFee)
			}

			if recipe.GetAmount().IsNeg() || recipe.GetAmount().IsZero() {
				recipe.IsValid = false
				recipe.Note = string(enums.INVALID_NOT_ENOUGH_BONDS_FOR_TX_FEES)
			}
			utils.AssertZAmountPositiveOrZero(recipe.GetAmount())
		}

		return recipe
	})

	ctx.StageData.AccumulatedPayouts = utils.OnlyValidAccumulatedPayouts(recipesWithEstimate) // overwrite with new only valid ones

	newInvalidAccumulatedRecipes := utils.OnlyInvalidAccumulatedPayouts(recipesWithEstimate)
	newInvalidRecipes := lo.Flatten(lo.Map(newInvalidAccumulatedRecipes, func(recipe *common.AccumulatedPayoutRecipe, _ int) []common.PayoutRecipe {
		return recipe.DisperseToInvalid()
	}))
	ctx.StageData.InvalidRecipes = append(ctx.StageData.InvalidRecipes, newInvalidRecipes...)
	return ctx, nil
}
