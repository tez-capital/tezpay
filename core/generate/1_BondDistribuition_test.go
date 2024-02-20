package generate

import (
	"testing"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/stretchr/testify/assert"
)

func TestGetBakerBondsAmount(t *testing.T) {
	assert := assert.New(t)

	configWithOverdelegationProtectionEnabled := configuration.GetDefaultRuntimeConfiguration()
	configWithOverdelegationProtectionDisabled := configuration.GetDefaultRuntimeConfiguration()
	configWithOverdelegationProtectionDisabled.Overdelegation.IsProtectionEnabled = false

	cycleData := common.BakersCycleData{
		OwnStakingBalance:        tezos.NewZ(500_000),
		OwnDelegatedBalance:      tezos.NewZ(500_000),
		ExternalDelegatedBalance: tezos.NewZ(19_000_000),
		BlockRewards:             tezos.NewZ(1000),
		EndorsementRewards:       tezos.NewZ(10000),
	}

	bakerBondsAmount := getBakerBondsAmount(&cycleData, tezos.NewZ(19_000_000), &configWithOverdelegationProtectionEnabled)
	assert.Equal(bakerBondsAmount, tezos.NewZ(1100))

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(19_000_000), &configWithOverdelegationProtectionDisabled)
	assert.Equal(bakerBondsAmount, tezos.NewZ(550))

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), &configWithOverdelegationProtectionEnabled)
	assert.Equal(bakerBondsAmount, tezos.NewZ(1100))

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), &configWithOverdelegationProtectionDisabled)
	assert.Equal(bakerBondsAmount, tezos.NewZ(1100))

	cycleData = common.BakersCycleData{
		OwnStakingBalance:        tezos.NewZ(600_000),
		OwnDelegatedBalance:      tezos.NewZ(400_000),
		ExternalDelegatedBalance: tezos.NewZ(9_000_000),
		BlockRewards:             tezos.NewZ(1000),
		EndorsementRewards:       tezos.NewZ(10000),
	}

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), &configWithOverdelegationProtectionEnabled)
	assert.Equal(bakerBondsAmount, tezos.NewZ(1100))

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), &configWithOverdelegationProtectionDisabled)
	assert.Equal(bakerBondsAmount, tezos.NewZ(1100))
}
