package execute

import (
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/utils"
)

type StageData struct {
	Limits                        *common.OperationLimits
	ReportsOfPastSuccesfulPayouts []common.PayoutReport
	Batches                       []common.RecipeBatch
	BatchResults                  common.BatchResults
	PaidDelegators                int
}

type PayoutExecutionContext struct {
	common.ExecutePayoutsEngineContext
	configuration *configuration.RuntimeConfiguration

	protectedSection *utils.ProtectedSection
	StageData        *StageData

	ValidPayouts       []common.PayoutRecipe
	InvalidPayouts     []common.PayoutRecipe
	AccumulatedPayouts []common.PayoutRecipe
	PayoutBlueprints   []*common.CyclePayoutBlueprint
}

func (ctx *PayoutExecutionContext) GetConfiguration() *configuration.RuntimeConfiguration {
	return ctx.configuration
}

func NewPayoutExecutionContext(preparationResult *common.PreparePayoutsResult, configuration *configuration.RuntimeConfiguration, engineContext *common.ExecutePayoutsEngineContext, options *common.ExecutePayoutsOptions) (*PayoutExecutionContext, error) {
	if err := engineContext.Validate(); err != nil {
		return nil, err
	}

	return &PayoutExecutionContext{
		ExecutePayoutsEngineContext: *engineContext,
		configuration:               configuration,

		protectedSection: utils.NewProtectedSection("executing payouts, job will be terminated after next batch"),
		StageData: &StageData{
			ReportsOfPastSuccesfulPayouts: preparationResult.ReportsOfPastSuccesfulPayouts,
		},

		ValidPayouts:       preparationResult.ValidPayouts,
		InvalidPayouts:     preparationResult.InvalidPayouts,
		AccumulatedPayouts: preparationResult.AccumulatedPayouts,
		PayoutBlueprints:   preparationResult.Blueprints,
	}, nil
}
