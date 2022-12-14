package payout

import (
	"fmt"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/payout/stages"
	log "github.com/sirupsen/logrus"
)

const (
	PAYOUT_EXECUTION_FAILURE = iota
	PAYOUT_EXECUTION_SUCCESS
)

func generatePayouts(payoutAddr tezos.Key, config *configuration.RuntimeConfiguration, options common.GeneratePayoutsOptions) (*common.CyclePayoutBlueprint, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration not specified")
	}

	ctx, err := stages.InitContext(payoutAddr, config, options)
	if err != nil {
		return nil, err
	}

	log.Infof("generating payouts for cycle %d (baker: '%s')", ctx.GetCycle(), config.BakerPKH)
	return ctx.Wrap().ExecuteStages(stages.GeneratePayoutCandidates,
		stages.DistributeBonds,
		stages.CheckSufficientBalance,
		stages.CollectBakerFee,
		stages.CollectTransactionFees,
		stages.ValidateSimulatedPayouts,
		stages.FinalizePayouts).ToCyclePayoutBlueprint()
}

func GeneratePayouts(payoutKey tezos.Key, config *configuration.RuntimeConfiguration, options common.GeneratePayoutsOptions) (*common.CyclePayoutBlueprint, error) {
	return generatePayouts(payoutKey, config, options)
}
