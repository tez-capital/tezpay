package configuration

import (
	"testing"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/stretchr/testify/assert"
)

func TestIsDonatingToTezCapital(t *testing.T) {
	assert := assert.New(t)
	configuration := GetDefaultRuntimeConfiguration()

	configuration.IncomeRecipients.Donate = .05
	configuration.IncomeRecipients.Donations = map[string]float64{
		constants.DEFAULT_DONATION_ADDRESS: .5,
	}
	assert.True(configuration.IsDonatingToTezCapital())

	configuration.IncomeRecipients.Donate = .0
	configuration.IncomeRecipients.Donations = map[string]float64{
		constants.DEFAULT_DONATION_ADDRESS: .5,
	}
	assert.False(configuration.IsDonatingToTezCapital())

	configuration.IncomeRecipients.Donate = .05
	configuration.IncomeRecipients.Donations = map[string]float64{
		tezos.ZeroAddress.String(): .5,
	}
	assert.True(configuration.IsDonatingToTezCapital())

	configuration.IncomeRecipients.Donate = .05
	configuration.IncomeRecipients.Donations = map[string]float64{
		tezos.ZeroAddress.String(): 1,
	}
	assert.False(configuration.IsDonatingToTezCapital())
}
