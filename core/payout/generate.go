package payout

import (
	"fmt"

	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/payout/generate"
)

const (
	PAYOUT_EXECUTION_FAILURE = iota
	PAYOUT_EXECUTION_SUCCESS
)

func GeneratePayouts(config *configuration.RuntimeConfiguration, engineContext *common.GeneratePayoutsEngineContext, options *common.GeneratePayoutsOptions) (*common.GeneratePayoutsResult, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration not specified")
	}

	ctx, err := generate.NewPayoutGenerationContext(config, engineContext)
	if err != nil {
		return nil, err
	}

	ctx, err = WrapContext[*generate.PayoutGenerationContext, *common.GeneratePayoutsOptions](ctx).ExecuteStages(options,
		generate.GeneratePayoutCandidates,
		generate.DistributeBonds,
		generate.CheckSufficientBalance,
		generate.CollectBakerFee,
		generate.CollectTransactionFees,
		generate.ValidateSimulatedPayouts,
		generate.FinalizePayouts,
		generate.CreateBlueprint).Unwrap()
	fmt.Println(ctx.StageData.PayoutBlueprint)
	return ctx.StageData.PayoutBlueprint, err
}
