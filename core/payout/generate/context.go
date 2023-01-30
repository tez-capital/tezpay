package generate

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/common"
	log "github.com/sirupsen/logrus"
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
}

type PayoutGenerationContext struct {
	common.GeneratePayoutsEngineContext
	PayoutKey            tezos.Key
	configuration        *configuration.RuntimeConfiguration
	StageData            StageData
	DistributableRewards tezos.Z
}

func NewPayoutGenerationContext(configuration *configuration.RuntimeConfiguration, engineContext *common.GeneratePayoutsEngineContext) (*PayoutGenerationContext, error) {
	log.Trace("tezpay payout context initialization")
	if err := engineContext.Validate(); err != nil {
		return nil, err
	}

	ctx := PayoutGenerationContext{
		GeneratePayoutsEngineContext: *engineContext,
		configuration:                configuration,
		StageData:                    StageData{},
		PayoutKey:                    engineContext.GetSigner().GetKey(),
	}

	return &ctx, nil
}

func (ctx *PayoutGenerationContext) GetConfiguration() *configuration.RuntimeConfiguration {
	return ctx.configuration
}

// func (ctx *PayoutGenerationContext) Wrap() WrappedStageResult {
// 	return WrappedStageResult{
// 		Ctx: ctx,
// 		Err: nil,
// 	}
// }

// func (ctx *PayoutGenerationContext) Run(stage WrappedStage, options *common.GeneratePayoutsOptions) WrappedStageResult {
// 	return stage(ctx.Wrap(), options)
// }
