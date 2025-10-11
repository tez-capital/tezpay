package generate

import (
	"log/slog"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/trilitech/tzgo/tezos"
)

type StageData struct {
	CycleData                             *common.BakersCycleData
	PayoutCandidates                      []PayoutCandidate
	PayoutCandidatesWithBondAmount        []PayoutCandidateWithBondAmount
	PayoutCandidatesWithBondAmountAndFees []PayoutCandidateWithBondAmountAndFee
	PayoutBlueprint                       *common.CyclePayoutBlueprint

	Payouts           []common.PayoutRecipe
	BakerBondsAmount  tezos.Z
	DonateBondsAmount tezos.Z
	// PaidDelegators    int
}

type PayoutGenerationContext struct {
	common.GeneratePayoutsEngineContext
	configuration *configuration.RuntimeConfiguration

	StageData *StageData

	PayoutKey tezos.Key

	logger *slog.Logger
}

func NewPayoutGenerationContext(configuration *configuration.RuntimeConfiguration, engineContext *common.GeneratePayoutsEngineContext) (*PayoutGenerationContext, error) {
	slog.Debug("tezpay payout context initialization")
	if err := engineContext.Validate(); err != nil {
		return nil, err
	}

	ctx := PayoutGenerationContext{
		GeneratePayoutsEngineContext: *engineContext,
		configuration:                configuration,

		StageData: &StageData{},

		PayoutKey: engineContext.GetSigner().GetKey(),

		logger: slog.Default().With("stage", "generate"),
	}

	return &ctx, nil
}

func (ctx *PayoutGenerationContext) GetConfiguration() *configuration.RuntimeConfiguration {
	return ctx.configuration
}
