package execute

import (
	"log/slog"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/utils"
)

type StageData struct {
	Limits                        *common.OperationLimits
	ReportsOfPastSuccesfulPayouts []common.PayoutReport
	Batches                       []common.RecipeBatch
	BatchResults                  common.BatchResults
	Summary                       common.PayoutSummary
}

type PayoutExecutionContext struct {
	common.ExecutePayoutsEngineContext
	configuration *configuration.RuntimeConfiguration

	protectedSection *utils.ProtectedSection
	StageData        *StageData

	ValidPayouts     []*common.AccumulatedPayoutRecipe
	InvalidPayouts   []common.PayoutRecipe
	PayoutBlueprints []*common.CyclePayoutBlueprint

	BatchMetadataDeserializationGasLimit int64

	logger *slog.Logger
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
			ReportsOfPastSuccesfulPayouts: preparationResult.ReportsOfPastSuccessfulPayouts,
		},

		ValidPayouts:     preparationResult.ValidPayouts,
		InvalidPayouts:   preparationResult.InvalidPayouts,
		PayoutBlueprints: preparationResult.Blueprints,

		BatchMetadataDeserializationGasLimit: preparationResult.BatchMetadataDeserializationGasLimit,

		logger: slog.Default().With("stage", "execute"),
	}, nil
}
