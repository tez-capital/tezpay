package payout

import (
	"fmt"

	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/payout/prepare"
)

func PreparePayouts(blueprint *common.GeneratePayoutsResult, config *configuration.RuntimeConfiguration, engineContext *common.PreparePayoutsEngineContext, options *common.PreparePayoutsOptions) (*common.PreparePayoutsResult, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration not specified")
	}

	ctx, err := prepare.NewPayoutPreparationContext(blueprint, config, engineContext, options)
	if err != nil {
		return nil, err
	}

	ctx, err = WrapContext[*prepare.PayoutPrepareContext, *common.PreparePayoutsOptions](ctx).ExecuteStages(options,
		prepare.PreparePayouts).Unwrap()
	return &common.PreparePayoutsResult{
		Blueprint:                     ctx.PayoutBlueprint,
		Payouts:                       ctx.StageData.Payouts,
		ReportsOfPastSuccesfulPayouts: ctx.StageData.ReportsOfPastSuccesfulPayouts,
	}, err
}
