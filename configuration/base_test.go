package configuration

import (
	"strings"
	"testing"

	"blockwatch.cc/tzgo/tezos"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/v"
	test_assert "github.com/stretchr/testify/assert"
)

func TestConfigurationToRuntimeConfiguration(t *testing.T) {
	assert := test_assert.New(t)
	runtime, _ := ConfigurationToRuntimeConfiguration(&LatestConfigurationType{
		Delegators: tezpay_configuration.DelegatorsConfigurationV0{
			FeeOverrides: map[string][]tezos.Address{
				".5": {tezos.InvalidAddress, tezos.BurnAddress},
				"1":  {tezos.ZeroAddress},
			},
		},
	})
	val, ok := runtime.Delegators.Overrides[tezos.InvalidAddress.String()]
	assert.True(ok)
	assert.Equal(*val.Fee, 0.5)

	val, ok = runtime.Delegators.Overrides[tezos.BurnAddress.String()]
	assert.True(ok)
	assert.Equal(*val.Fee, 0.5)

	val, ok = runtime.Delegators.Overrides[tezos.ZeroAddress.String()]
	assert.True(ok)
	assert.Equal(*val.Fee, float64(1))

	runtime, _ = ConfigurationToRuntimeConfiguration(&LatestConfigurationType{
		Delegators: tezpay_configuration.DelegatorsConfigurationV0{
			FeeOverrides: map[string][]tezos.Address{
				"0": {tezos.InvalidAddress, tezos.BurnAddress},
			},
		},
	})

	val, ok = runtime.Delegators.Overrides[tezos.InvalidAddress.String()]
	assert.True(ok)
	assert.Equal(*val.Fee, 0.)

	val, ok = runtime.Delegators.Overrides[tezos.BurnAddress.String()]
	assert.True(ok)
	assert.Equal(*val.Fee, 0.)

	fee := 1.0
	runtime, _ = ConfigurationToRuntimeConfiguration(&LatestConfigurationType{
		Delegators: tezpay_configuration.DelegatorsConfigurationV0{
			FeeOverrides: map[string][]tezos.Address{
				"0": {tezos.InvalidAddress, tezos.BurnAddress},
			},
			Overrides: map[string]tezpay_configuration.DelegatorOverrideV0{
				tezos.InvalidAddress.String(): {
					Fee: &fee,
				},
			},
		},
	})

	val, ok = runtime.Delegators.Overrides[tezos.InvalidAddress.String()]
	assert.True(ok)
	assert.Equal(*val.Fee, float64(1))

	val, ok = runtime.Delegators.Overrides[tezos.BurnAddress.String()]
	assert.True(ok)
	assert.Equal(*val.Fee, 0.)

	runtime, _ = ConfigurationToRuntimeConfiguration(&LatestConfigurationType{
		Delegators: tezpay_configuration.DelegatorsConfigurationV0{
			FeeOverrides: map[string][]tezos.Address{
				"1.1": {tezos.InvalidAddress, tezos.BurnAddress},
			},
		},
	})

	err := runtime.Validate()
	assert.NotNil(err)
	assert.True(strings.Contains(err.Error(), "fee must be between 0 and 1"))
}
