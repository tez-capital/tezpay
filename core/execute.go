package core

import (
	"fmt"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/execute"
)

func ExecutePayouts(preparationResult *common.PreparePayoutsResult, config *configuration.RuntimeConfiguration, engineContext *common.ExecutePayoutsEngineContext, options *common.ExecutePayoutsOptions) (common.ExecutePayoutsResult, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration not specified")
	}

	ctx, err := execute.NewPayoutExecutionContext(preparationResult, config, engineContext, options)
	if err != nil {
		return nil, err
	}

	ctx, err = WrapContext[*execute.PayoutExecutionContext, *common.ExecutePayoutsOptions](ctx).ExecuteStages(options,
		execute.SplitIntoBatches,
		execute.ExecutePayouts).Unwrap()
	return ctx.StageData.BatchResults, err
}
