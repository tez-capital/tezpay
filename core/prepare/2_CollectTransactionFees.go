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
	recipesAfterEstimate := ctx.StageData.AccumulatedValidPayouts
	logger := ctx.logger.With("phase", "collect_transaction_fees")
	logger.Info("collecting transaction fees")

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
	recipesAfterEstimate = lo.Map(estimate.EstimateTransactionFees(recipesAfterEstimate, estimateContext), func(result estimate.EstimateResult[*common.AccumulatedPayoutRecipe], _ int) *common.AccumulatedPayoutRecipe {
		if result.Error != nil {
			slog.Warn("failed to estimate tx costs", "recipient", result.Transaction.Recipient, "delegator", payoutKey.Address(), "amount", result.Transaction.Amount.Int64(), "kind", result.Transaction.TxKind, "error", result.Error)
			result.Transaction.IsValid = false
			result.Transaction.Note = string(enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS)
			return result.Transaction
		}

		recipe := result.Transaction

		if recipe.TxKind == enums.PAYOUT_TX_KIND_TEZ {
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

			bondsAmountBeforeFees := recipe.Amount
			utils.AssertZAmountPositiveOrZero(bondsAmountBeforeFees)

			txFee := result.OpLimits.GetOperationFeesWithoutAllocation()
			allocationFee := result.OpLimits.GetAllocationFee()
			if !isBakerPayingTxFee {
				recipe.Amount = recipe.Amount.Sub64(txFee)
			}
			if !isBakerPayingAllocationTxFee {
				recipe.Amount = recipe.Amount.Sub64(allocationFee)
			}

			if recipe.Amount.IsNeg() || recipe.Amount.IsZero() {
				recipe.IsValid = false
				recipe.Note = string(enums.INVALID_NOT_ENOUGH_BONDS_FOR_TX_FEES)
			}
			utils.AssertZAmountPositiveOrZero(recipe.Amount)
		}

		result.Transaction.OpLimits = result.OpLimits
		return result.Transaction
	})

	validRecipes := utils.OnlyValidAccumulatedPayouts(recipesAfterEstimate)
	invalidAccumulatedRecipes := utils.OnlyInvalidAccumulatedPayouts(recipesAfterEstimate)
	invalidRecipes := lo.Map(invalidAccumulatedRecipes, func(recipe *common.AccumulatedPayoutRecipe, _ int) []common.PayoutRecipe {
		return recipe.DisperseToInvalid()
	})

	ctx.StageData.AccumulatedValidPayouts = validRecipes
	ctx.StageData.InvalidPayouts = append(ctx.StageData.InvalidPayouts, lo.Flatten(invalidRecipes)...)
	return ctx, nil
}
