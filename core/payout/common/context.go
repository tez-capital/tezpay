package common

import (
	"fmt"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/clients"
	"github.com/alis-is/tezpay/clients/interfaces"
	tezpay_tezos "github.com/alis-is/tezpay/clients/tezos"
	"github.com/alis-is/tezpay/configuration"
	log "github.com/sirupsen/logrus"
)

type StageData struct {
	PayoutCandidates                      []PayoutCandidate
	PayoutCandidatesWithBondAmount        []PayoutCandidateWithBondAmount
	PayoutCandidatesWithBondAmountAndFees []PayoutCandidateWithBondAmountAndFee
	PayoutCandidatesSimulated             []PayoutCandidateSimulated

	Payouts           []PayoutRecipe
	BakerBondsAmount  tezos.Z
	DonateBondsAmount tezos.Z
	BakerFeesAmount   tezos.Z
	DonateFeesAmount  tezos.Z
}

type Context struct {
	PayoutKey            tezos.Key
	configuration        *configuration.RuntimeConfiguration
	Collector            interfaces.CollectorEngine
	Cycle                int64
	CycleData            *tezpay_tezos.BakersCycleData
	StageData            StageData
	DistributableRewards tezos.Z
}

func InitContext(payoutKey tezos.Key, configuration *configuration.RuntimeConfiguration, cycle int64) (*Context, error) {
	log.Debug("tezpay engine initialization")
	collector, err := clients.InitDefaultRpcAndTzktColletor(configuration.Network.RpcUrl, configuration.Network.TzktUrl)
	if err != nil {
		return nil, err
	}

	if cycle == 0 {
		cycle, err = collector.GetLastCompletedCycle()
		if err != nil {
			return nil, err
		}
	}

	log.Infof("collecting rewards split through %s collector", collector.GetId())
	cycleData, err := collector.GetCycleData(configuration.BakerPKH, cycle)
	if err != nil {
		return nil, fmt.Errorf("failed to collect cycle data through collector %s - %s", collector.GetId(), err.Error())
	}

	ctx := Context{
		configuration: configuration,
		Collector:     collector,
		CycleData:     cycleData,
		Cycle:         cycle,
		StageData:     StageData{},
		PayoutKey:     payoutKey,
	}

	return &ctx, nil
}

func (ctx *Context) GetConfiguration() *configuration.RuntimeConfiguration {
	return ctx.configuration
}

func (ctx *Context) GetCycleData() *tezpay_tezos.BakersCycleData {
	return ctx.CycleData
}

func (ctx *Context) GetCycle() int64 {
	return ctx.Cycle
}

func (ctx *Context) GetCollector() interfaces.CollectorEngine {
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
