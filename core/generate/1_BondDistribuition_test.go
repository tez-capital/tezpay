package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/trilitech/tzgo/tezos"
)

func TestGetBakerBondsAmount(t *testing.T) {
	assert := assert.New(t)

	configWithOverdelegationProtectionEnabled := configuration.GetDefaultRuntimeConfiguration()
	configWithOverdelegationProtectionDisabled := configuration.GetDefaultRuntimeConfiguration()
	configWithOverdelegationProtectionDisabled.Overdelegation.IsProtectionEnabled = false

	cycleData := common.BakersCycleData{
		OwnStakedBalance:            tezos.NewZ(500_000),
		OwnDelegatedBalance:         tezos.NewZ(500_000),
		ExternalDelegatedBalance:    tezos.NewZ(19_000_000),
		BlockDelegatedRewards:       tezos.NewZ(1000),
		EndorsementDelegatedRewards: tezos.NewZ(10000),
	}

	bakerBondsAmount := getBakerBondsAmount(&cycleData, tezos.NewZ(19_000_000), &configWithOverdelegationProtectionEnabled)
	assert.Equal(bakerBondsAmount.Int64(), tezos.NewZ(1222).Int64())

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(19_000_000), &configWithOverdelegationProtectionDisabled)
	assert.Equal(bakerBondsAmount.Int64(), tezos.NewZ(282).Int64())

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), &configWithOverdelegationProtectionEnabled)
	assert.Equal(bakerBondsAmount.Int64(), tezos.NewZ(1222).Int64())

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), &configWithOverdelegationProtectionDisabled)
	assert.Equal(bakerBondsAmount.Int64(), tezos.NewZ(578).Int64())

	cycleData = common.BakersCycleData{
		OwnStakedBalance:            tezos.NewZ(600_000),
		OwnDelegatedBalance:         tezos.NewZ(400_000),
		ExternalDelegatedBalance:    tezos.NewZ(9_000_000),
		BlockDelegatedRewards:       tezos.NewZ(1000),
		EndorsementDelegatedRewards: tezos.NewZ(10000),
	}

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), &configWithOverdelegationProtectionEnabled)
	assert.Equal(bakerBondsAmount.Int64(), tezos.NewZ(814).Int64())

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), &configWithOverdelegationProtectionDisabled)
	assert.Equal(bakerBondsAmount.Int64(), tezos.NewZ(468).Int64())
}
