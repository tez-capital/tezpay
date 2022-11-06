package configuration

import (
	"errors"
	"fmt"

	"blockwatch.cc/tzgo/tezos"
	"github.com/samber/lo"
)

func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}

func getPortionRangeError(id string, value float64) string {
	return fmt.Sprintf("%s must be between 0 and 1. Current value '%.2f'", id, value)
}

func (configuration *RuntimeConfiguration) Validate() (err error) {
	defer func() {
		msg, _ := recover().(string)
		if msg != "" {
			err = errors.New(msg)
		}
	}()

	assert(configuration.PayoutConfiguration.Fee >= 0 && configuration.PayoutConfiguration.Fee <= 1,
		getPortionRangeError("configuration.payouts.fee", configuration.PayoutConfiguration.Fee))
	assert(configuration.IncomeRecipients.Donate >= 0 && configuration.PayoutConfiguration.Fee <= 1,
		getPortionRangeError("configuration.income_recipients.donate", configuration.PayoutConfiguration.Fee))

	bondsPortions := lo.Reduce(lo.Values(configuration.IncomeRecipients.Bonds), func(agg float64, val float64, _ int) float64 {
		return agg + val
	}, float64(0))
	assert(bondsPortions >= 0 && bondsPortions <= 1,
		getPortionRangeError("configuration.income_recipients.bonds sum", bondsPortions))
	for k := range configuration.IncomeRecipients.Bonds {
		_, err := tezos.ParseAddress(k)
		assert(err == nil, fmt.Sprintf("configuration.income_recipients.bonds.%s has to be valid PKH", k))
	}

	feesPortions := lo.Reduce(lo.Values(configuration.IncomeRecipients.Fees), func(agg float64, val float64, _ int) float64 {
		return agg + val
	}, float64(0))
	assert(feesPortions >= 0 && feesPortions <= 1,
		getPortionRangeError("configuration.income_recipients.fees sum", feesPortions))
	for k := range configuration.IncomeRecipients.Fees {
		_, err := tezos.ParseAddress(k)
		assert(err == nil, fmt.Sprintf("configuration.income_recipients.fees.%s has to be valid PKH", k))
	}

	donatePortions := lo.Reduce(lo.Values(configuration.IncomeRecipients.Donations), func(agg float64, val float64, _ int) float64 {
		return agg + val
	}, float64(0))
	assert(donatePortions >= 0 && donatePortions <= 1,
		getPortionRangeError("configuration.income_recipients.donations sum", donatePortions))
	for k := range configuration.IncomeRecipients.Donations {
		_, err := tezos.ParseAddress(k)
		assert(err == nil, fmt.Sprintf("configuration.income_recipients.donations.%s has to be valid PKH", k))
	}

	for k, v := range configuration.Delegators.Overrides {
		_, err := tezos.ParseAddress(k)
		assert(err == nil, fmt.Sprintf("configuration.delegators.overrides.%s has to be valid PKH", k))
		assert(v.Fee >= 0 && v.Fee <= 1,
			getPortionRangeError(fmt.Sprintf("configuration.delegators.overrides.%s fee", k), v.Fee))
	}

	return
}
