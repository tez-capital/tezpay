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
	common.PreparePayoutsEngineContext
	configuration *configuration.RuntimeConfiguration

	StageData *StageData

	PayoutBlueprint *common.CyclePayoutBlueprint
}

func (ctx *PayoutPrepareContext) GetConfiguration() *configuration.RuntimeConfiguration {
	return ctx.configuration
}

func NewPayoutPreparationContext(blueprint *common.CyclePayoutBlueprint, configuration *configuration.RuntimeConfiguration, engineContext *common.PreparePayoutsEngineContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	if err := engineContext.Validate(); err != nil {
		return nil, err
	}

	return &PayoutPrepareContext{
		PreparePayoutsEngineContext: *engineContext,
		configuration:               configuration,

		StageData: &StageData{},

		PayoutBlueprint: blueprint,
	}, nil
}
