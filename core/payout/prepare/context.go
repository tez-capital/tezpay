package prepare

import (
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/common"
)

type StageData struct {
	Payouts                       []common.PayoutRecipe
	ReportsOfPastSuccesfulPayouts []common.PayoutReport
}

type PayoutPrepareContext struct {
	common.PreparePayoutsOptions
	common.PreparePayoutsEngineContext

	StageData StageData

	PayoutBlueprint *common.CyclePayoutBlueprint
	configuration   *configuration.RuntimeConfiguration
}

func (ctx *PayoutPrepareContext) GetConfiguration() *configuration.RuntimeConfiguration {
	return ctx.configuration
}

func NewPayoutPreparationContext(blueprint *common.CyclePayoutBlueprint, configuration *configuration.RuntimeConfiguration, engineContext *common.PreparePayoutsEngineContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	if err := engineContext.Validate(); err != nil {
		return nil, err
	}

	return &PayoutPrepareContext{
		PayoutBlueprint:             blueprint,
		PreparePayoutsOptions:       *options,
		PreparePayoutsEngineContext: *engineContext,
		configuration:               configuration,
	}, nil
}
