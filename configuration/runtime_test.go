package configuration

import (
	"testing"

	"github.com/alis-is/tezpay/constants"
	"github.com/stretchr/testify/assert"
	"github.com/trilitech/tzgo/tezos"
)

func TestIsDonatingToTezCapital(t *testing.T) {
	assert := assert.New(t)
	configuration := GetDefaultRuntimeConfiguration()

	configuration.IncomeRecipients.DonateBonds = .05
	configuration.IncomeRecipients.DonateFees = .05
	configuration.IncomeRecipients.Donations = map[string]float64{
		constants.DEFAULT_DONATION_ADDRESS: .5,
	}
	assert.True(configuration.IsDonatingToTezCapital())

	configuration.IncomeRecipients.DonateFees = .0
	configuration.IncomeRecipients.DonateBonds = .0
	configuration.IncomeRecipients.Donations = map[string]float64{
		constants.DEFAULT_DONATION_ADDRESS: .5,
	}
	assert.False(configuration.IsDonatingToTezCapital())

	configuration.IncomeRecipients.DonateBonds = .05
	configuration.IncomeRecipients.DonateFees = .05
	configuration.IncomeRecipients.Donations = map[string]float64{
		tezos.ZeroAddress.String(): .5,
	}
	assert.True(configuration.IsDonatingToTezCapital())

	configuration.IncomeRecipients.DonateBonds = .05
	configuration.IncomeRecipients.DonateFees = .05
	configuration.IncomeRecipients.Donations = map[string]float64{
		tezos.ZeroAddress.String(): 1,
	}
	assert.False(configuration.IsDonatingToTezCapital())
}
