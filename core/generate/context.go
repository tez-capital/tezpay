package generate

import (
	log "github.com/sirupsen/logrus"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/trilitech/tzgo/tezos"
)

type StageData struct {
	CycleData                             *common.BakersCycleData
	PayoutCandidates                      []PayoutCandidate
	PayoutCandidatesWithBondAmount        []PayoutCandidateWithBondAmount
	PayoutCandidatesWithBondAmountAndFees []PayoutCandidateWithBondAmountAndFee
	PayoutCandidatesSimulated             []PayoutCandidateSimulated
	PayoutBlueprint                       *common.CyclePayoutBlueprint

	Payouts           []common.PayoutRecipe
	BakerBondsAmount  tezos.Z
	DonateBondsAmount tezos.Z
	BakerFeesAmount   tezos.Z
	DonateFeesAmount  tezos.Z
	PaidDelegators    int

	// protocol, signature etc.
	BatchMetadataDeserializationGasLimit int64
}

type PayoutGenerationContext struct {
	common.GeneratePayoutsEngineContext
	configuration *configuration.RuntimeConfiguration

	StageData *StageData

	PayoutKey tezos.Key
}

func NewPayoutGenerationContext(configuration *configuration.RuntimeConfiguration, engineContext *common.GeneratePayoutsEngineContext) (*PayoutGenerationContext, error) {
	log.Trace("tezpay payout context initialization")
	if err := engineContext.Validate(); err != nil {
		return nil, err
	}

	ctx := PayoutGenerationContext{
		GeneratePayoutsEngineContext: *engineContext,
		configuration:                configuration,

		StageData: &StageData{},

		PayoutKey: engineContext.GetSigner().GetKey(),
	}

	return &ctx, nil
}

func (ctx *PayoutGenerationContext) GetConfiguration() *configuration.RuntimeConfiguration {
	return ctx.configuration
}
