package stages

import (
	"errors"
	"fmt"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/common"
	log "github.com/sirupsen/logrus"
)

type StageData struct {
	PayoutCandidates                      []PayoutCandidate
	PayoutCandidatesWithBondAmount        []PayoutCandidateWithBondAmount
	PayoutCandidatesWithBondAmountAndFees []PayoutCandidateWithBondAmountAndFee
	PayoutCandidatesSimulated             []PayoutCandidateSimulated

	Payouts           []common.PayoutRecipe
	BakerBondsAmount  tezos.Z
	DonateBondsAmount tezos.Z
	BakerFeesAmount   tezos.Z
	DonateFeesAmount  tezos.Z
	PaidDelegators    int
}

type Context struct {
	PayoutKey            tezos.Key
	configuration        *configuration.RuntimeConfiguration
	Collector            common.CollectorEngine
	Cycle                int64
	CycleData            *common.BakersCycleData
	StageData            StageData
	DistributableRewards tezos.Z
	Options              common.GeneratePayoutsOptions
}

func InitContext(payoutKey tezos.Key, configuration *configuration.RuntimeConfiguration, options common.GeneratePayoutsOptions) (*Context, error) {
	log.Trace("tezpay payout context initialization")
	if options.Engines.Collector == nil {
		return nil, errors.New("udefined collector engine")
	}
	collector := options.Engines.Collector

	if options.Cycle == 0 {
		cycle, err := collector.GetLastCompletedCycle()
		if err != nil {
			return nil, err
		}
		options.Cycle = cycle
	}

	log.Infof("collecting rewards split through %s collector", collector.GetId())
	cycleData, err := collector.GetCycleData(configuration.BakerPKH, options.Cycle)
	if err != nil {
		return nil, fmt.Errorf("failed to collect cycle data through collector %s - %s", collector.GetId(), err.Error())
	}

	ctx := Context{
		configuration: configuration,
		Collector:     collector,
		CycleData:     cycleData,
		Cycle:         options.Cycle,
		StageData:     StageData{},
		PayoutKey:     payoutKey,
		Options:       options,
	}

	return &ctx, nil
}

func (ctx *Context) GetConfiguration() *configuration.RuntimeConfiguration {
	return ctx.configuration
}

func (ctx *Context) GetCycleData() *common.BakersCycleData {
	return ctx.CycleData
}

func (ctx *Context) GetCycle() int64 {
	return ctx.Cycle
}

func (ctx *Context) GetCollector() common.CollectorEngine {
	return ctx.Collector
}

func (ctx *Context) Wrap() WrappedStageResult {
	return WrappedStageResult{
		Ctx: *ctx,
		Err: nil,
	}
}

func (ctx *Context) Run(stage WrappedStage) WrappedStageResult {
	return stage(ctx.Wrap())
}
