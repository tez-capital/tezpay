package core

import (
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/core/execute"
)

func ExecutePayouts(preparationResult *common.PreparePayoutsResult, config *configuration.RuntimeConfiguration, engineContext *common.ExecutePayoutsEngineContext, options *common.ExecutePayoutsOptions) (*common.ExecutePayoutsResult, error) {
	if config == nil {
		return nil, constants.ErrMissingConfiguration
	}

	ctx, err := execute.NewPayoutExecutionContext(preparationResult, config, engineContext, options)
	if err != nil {
		return nil, err
	}

	ctx, err = WrapContext[*execute.PayoutExecutionContext, *common.ExecutePayoutsOptions](ctx).ExecuteStages(options,
		execute.SplitIntoBatches,
		execute.ExecutePayouts).Unwrap()
	if err != nil {
		return nil, err
	}

	return &common.ExecutePayoutsResult{
		BatchResults: ctx.StageData.BatchResults,
		Summary:      ctx.StageData.Summary,
	}, nil
}
