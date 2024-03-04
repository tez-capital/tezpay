package prepare

import (
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
)

type StageData struct {
	ValidPayouts                  []common.PayoutRecipe
	InvalidPayouts                []common.PayoutRecipe
	AccumulatedPayouts            []common.PayoutRecipe
	ReportsOfPastSuccesfulPayouts []common.PayoutReport
}

type PayoutPrepareContext struct {
	common.PreparePayoutsEngineContext
	configuration *configuration.RuntimeConfiguration

	StageData *StageData

	PayoutBlueprints []*common.CyclePayoutBlueprint
}

func (ctx *PayoutPrepareContext) GetConfiguration() *configuration.RuntimeConfiguration {
	return ctx.configuration
}

func NewPayoutPreparationContext(blueprints []*common.CyclePayoutBlueprint, configuration *configuration.RuntimeConfiguration, engineContext *common.PreparePayoutsEngineContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	if err := engineContext.Validate(); err != nil {
		return nil, err
	}

	return &PayoutPrepareContext{
		PreparePayoutsEngineContext: *engineContext,
		configuration:               configuration,

		StageData: &StageData{},

		PayoutBlueprints: blueprints,
	}, nil
}
