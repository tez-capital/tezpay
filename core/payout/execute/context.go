package execute

import (
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/utils"
)

type StageData struct {
	Limits                        *common.OperationLimits
	ReportsOfPastSuccesfulPayouts []common.PayoutReport
	Batches                       []common.RecipeBatch
	BatchResults                  common.BatchResults
}

type PayoutExecutionContext struct {
	common.ExecutePayoutsOptions
	common.ExecutePayoutsEngineContext

	StageData *StageData

	Payouts          []common.PayoutRecipe
	PayoutBlueprint  *common.CyclePayoutBlueprint
	configuration    *configuration.RuntimeConfiguration
	protectedSection *utils.ProtectedSection
}

func (ctx *PayoutExecutionContext) GetConfiguration() *configuration.RuntimeConfiguration {
	return ctx.configuration
}

func NewPayoutExecutionContext(preparationResult *common.PreparePayoutsResult, configuration *configuration.RuntimeConfiguration, engineContext *common.ExecutePayoutsEngineContext, options *common.ExecutePayoutsOptions) (*PayoutExecutionContext, error) {
	if err := engineContext.Validate(); err != nil {
		return nil, err
	}

	return &PayoutExecutionContext{
		PayoutBlueprint:             preparationResult.Blueprint,
		Payouts:                     preparationResult.Payouts,
		ExecutePayoutsOptions:       *options,
		ExecutePayoutsEngineContext: *engineContext,
		configuration:               configuration,
		protectedSection:            utils.NewProtectedSection("executing payouts, job will be terminated after next batch"),
	}, nil
}
