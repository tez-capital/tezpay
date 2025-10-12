package prepare

import (
	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/utils"
)

func ValidatePreparedPayouts(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (result *PayoutPrepareContext, err error) {
	configuration := ctx.GetConfiguration()
	logger := ctx.logger.With("phase", "validate_prepared_payouts")
	accumulatedPayouts := ctx.StageData.AccumulatedPayouts

	logger.Info("validating prepared payout candidates")

	accumulatedPayouts = lo.Map(accumulatedPayouts, func(recipe *common.AccumulatedPayoutRecipe, _ int) *common.AccumulatedPayoutRecipe {
		if !recipe.IsValid {
			return recipe
		}

		if recipe.Kind != enums.PAYOUT_KIND_DELEGATOR_REWARD { // only delegator rewards are subject to validation
			return recipe
		}

		validationContext := createValidationContext(recipe, configuration)
		result := validationContext.Validate(
			MinumumAmountValidator,
		).Unwrap()

		utils.AssertZAmountPositiveOrZero(recipe.Amount)
		return result
	})

	ctx.StageData.AccumulatedPayouts = utils.OnlyValidAccumulatedPayouts(accumulatedPayouts)
	invalidAccumulatedRecipes := utils.OnlyInvalidAccumulatedPayouts(accumulatedPayouts)
	ctx.StageData.InvalidRecipes = append(ctx.StageData.InvalidRecipes, lo.FlatMap(invalidAccumulatedRecipes, func(recipe *common.AccumulatedPayoutRecipe, _ int) []common.PayoutRecipe {
		return recipe.DisperseToInvalid()
	})...)

	return ctx, nil
}
