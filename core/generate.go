package core

import (
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/core/generate"
)

const (
	PAYOUT_EXECUTION_FAILURE = iota
	PAYOUT_EXECUTION_SUCCESS
)

func GeneratePayouts(config *configuration.RuntimeConfiguration, engineContext *common.GeneratePayoutsEngineContext, options *common.GeneratePayoutsOptions) (*common.CyclePayoutBlueprint, error) {
	if config == nil {
		return nil, constants.ErrMissingConfiguration
	}

	ctx, err := generate.NewPayoutGenerationContext(config, engineContext)
	if err != nil {
		return nil, err
	}

	ctx, err = WrapContext[*generate.PayoutGenerationContext, *common.GeneratePayoutsOptions](ctx).ExecuteStages(options,
		generate.SendAnalytics,
		generate.CheckConditionsAndPrepare,
		generate.GeneratePayoutCandidates,
		// hooks
		generate.DistributeBonds,
		generate.CollectBakerFee,
		generate.ValidateRecipe,
		generate.FinalizeRecipes,
		generate.CreateBlueprint).Unwrap()
	if err != nil {
		return nil, err
	}

	return ctx.StageData.PayoutBlueprint, nil
}
