package generate

import (
	"testing"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
	"github.com/stretchr/testify/assert"
)

func TestGetBakerBondsAmount(t *testing.T) {
	assert := assert.New(t)

	cycleData := common.BakersCycleData{
		StakingBalance:     tezos.NewZ(20_000_000),
		DelegatedBalance:   tezos.NewZ(19_000_000),
		BlockRewards:       tezos.NewZ(1000),
		EndorsementRewards: tezos.NewZ(10000),
	}

	bakerBondsAmount := getBakerBondsAmount(&cycleData, tezos.NewZ(19_000_000), true)
	assert.Equal(bakerBondsAmount, tezos.NewZ(1100))

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(19_000_000), false)
	assert.Equal(bakerBondsAmount, tezos.NewZ(550))

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), true)
	assert.Equal(bakerBondsAmount, tezos.NewZ(1100))

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), false)
	assert.Equal(bakerBondsAmount, tezos.NewZ(1100))

	cycleData = common.BakersCycleData{
		StakingBalance:     tezos.NewZ(10_000_000),
		DelegatedBalance:   tezos.NewZ(9_000_000),
		BlockRewards:       tezos.NewZ(1000),
		EndorsementRewards: tezos.NewZ(10000),
	}

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), true)
	assert.Equal(bakerBondsAmount, tezos.NewZ(1100))

	bakerBondsAmount = getBakerBondsAmount(&cycleData, tezos.NewZ(9_000_000), false)
	assert.Equal(bakerBondsAmount, tezos.NewZ(1100))
}
