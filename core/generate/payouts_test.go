package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/trilitech/tzgo/tezos"
)

func TestDelegatorToPayoutCandidate(t *testing.T) {
	assert := assert.New(t)

	config := configuration.GetDefaultRuntimeConfiguration()

	maximumBalance := tezos.NewZ(100000000)
	config.Delegators.Overrides = map[string]configuration.RuntimeDelegatorOverride{
		"tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM": {
			MaximumBalance: &maximumBalance,
		},
		"tz1hZvgjekGo7DmQjWh7XnY5eLQD8wNYPczE": {
			MaximumBalance: &maximumBalance,
		},
	}

	delegators := []common.Delegator{
		{
			Address:          tezos.MustParseAddress("tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM"),
			DelegatedBalance: tezos.NewZ(100000000),
		},
		{
			Address:          tezos.MustParseAddress("tz1hZvgjekGo7DmQjWh7XnY5eLQD8wNYPczE"),
			DelegatedBalance: tezos.NewZ(200000000),
		},
	}

	delegator := delegators[0]
	candidate := DelegatorToPayoutCandidate(delegator, &config)
	assert.True(candidate.GetEffectiveBalance().Equal(tezos.MinZ(delegator.DelegatedBalance, maximumBalance)))

	delegator = delegators[1]
	candidate = DelegatorToPayoutCandidate(delegator, &config)
	assert.True(candidate.GetEffectiveBalance().Equal(tezos.MinZ(delegator.DelegatedBalance, maximumBalance)))

	config.Delegators.Overrides = map[string]configuration.RuntimeDelegatorOverride{}

	delegator = delegators[0]
	candidate = DelegatorToPayoutCandidate(delegator, &config)
	assert.True(candidate.GetEffectiveBalance().Equal(delegator.DelegatedBalance))

	delegator = delegators[1]
	candidate = DelegatorToPayoutCandidate(delegator, &config)
	assert.True(candidate.GetEffectiveBalance().Equal(delegator.DelegatedBalance))
}
