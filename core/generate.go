package core

import (
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core/generate"
)

const (
	PAYOUT_EXECUTION_FAILURE = iota
	PAYOUT_EXECUTION_SUCCESS
)

func GeneratePayouts(config *configuration.RuntimeConfiguration, engineContext *common.GeneratePayoutsEngineContext, options *common.GeneratePayoutsOptions) (*common.GeneratePayoutsResult, error) {
	if config == nil {
		return nil, constants.ErrMissingConfiguration
	}

	ctx, err := generate.NewPayoutGenerationContext(config, engineContext)
	if err != nil {
		return nil, err
	}

	ctx, err = WrapContext[*generate.PayoutGenerationContext, *common.GeneratePayoutsOptions](ctx).ExecuteStages(options,
		generate.SendAnalytics,
		generate.CheckConditions,
		generate.GeneratePayoutCandidates,
		// hooks
		generate.DistributeBonds,
		generate.CheckSufficientBalance,
		generate.CollectBakerFee,
		generate.CollectTransactionFees,
		generate.ValidateSimulatedPayouts,
		generate.FinalizePayouts,
		generate.CreateBlueprint).Unwrap()
	return ctx.StageData.PayoutBlueprint, err
}
