package core

import (
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/core/prepare"
)

func PreparePayouts(blueprints []*common.CyclePayoutBlueprint, config *configuration.RuntimeConfiguration, engineContext *common.PreparePayoutsEngineContext, options *common.PreparePayoutsOptions) (*common.PreparePayoutsResult, error) {
	if config == nil {
		return nil, constants.ErrMissingConfiguration
	}

	ctx, err := prepare.NewPayoutPreparationContext(blueprints, config, engineContext, options)
	if err != nil {
		return nil, err
	}

	ctx, err = WrapContext[*prepare.PayoutPrepareContext, *common.PreparePayoutsOptions](ctx).ExecuteStages(options,
		prepare.PreparePayouts,
		prepare.AccumulatePayouts).Unwrap()
	if err != nil {
		return nil, err
	}

	return &common.PreparePayoutsResult{
		Blueprints:                     ctx.PayoutBlueprints,
		ValidPayouts:                   ctx.StageData.ValidPayouts,
		AccumulatedPayouts:             ctx.StageData.AccumulatedPayouts,
		InvalidPayouts:                 ctx.StageData.InvalidPayouts,
		ReportsOfPastSuccessfulPayouts: ctx.StageData.ReportsOfPastSuccesfulPayouts,
	}, nil
}

func PrepareCyclePayouts(blueprint *common.CyclePayoutBlueprint, config *configuration.RuntimeConfiguration, engineContext *common.PreparePayoutsEngineContext, options *common.PreparePayoutsOptions) (*common.PreparePayoutsResult, error) {
	return PreparePayouts([]*common.CyclePayoutBlueprint{blueprint}, config, engineContext, options)
}
