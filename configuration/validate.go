package configuration

import (
	"errors"
	"fmt"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/notifications"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
)

func _assert(condition bool, msg string) {
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

	_assert(utils.IsPortionWithin0n1(configuration.PayoutConfiguration.Fee),
		getPortionRangeError("configuration.payouts.fee", configuration.PayoutConfiguration.Fee))
	_assert(utils.IsPortionWithin0n1(configuration.IncomeRecipients.Donate),
		getPortionRangeError("configuration.income_recipients.donate", configuration.IncomeRecipients.Donate))

	bondsPortions := lo.Reduce(lo.Values(configuration.IncomeRecipients.Bonds), func(agg float64, val float64, _ int) float64 {
		return agg + val
	}, float64(0))
	_assert(utils.IsPortionWithin0n1(bondsPortions), getPortionRangeError("configuration.income_recipients.bonds sum", bondsPortions))
	for k := range configuration.IncomeRecipients.Bonds {
		_, err := tezos.ParseAddress(k)
		_assert(err == nil, fmt.Sprintf("configuration.income_recipients.bonds.%s has to be valid PKH", k))
	}

	feesPortions := lo.Reduce(lo.Values(configuration.IncomeRecipients.Fees), func(agg float64, val float64, _ int) float64 {
		return agg + val
	}, float64(0))
	_assert(utils.IsPortionWithin0n1(feesPortions),
		getPortionRangeError("configuration.income_recipients.fees sum", feesPortions))
	for k := range configuration.IncomeRecipients.Fees {
		_, err := tezos.ParseAddress(k)
		_assert(err == nil, fmt.Sprintf("configuration.income_recipients.fees.%s has to be valid PKH", k))
	}

	donatePortions := lo.Reduce(lo.Values(configuration.IncomeRecipients.Donations), func(agg float64, val float64, _ int) float64 {
		return agg + val
	}, float64(0))
	_assert(utils.IsPortionWithin0n1(donatePortions),
		getPortionRangeError("configuration.income_recipients.donations sum", donatePortions))
	for k := range configuration.IncomeRecipients.Donations {
		_, err := tezos.ParseAddress(k)
		_assert(err == nil, fmt.Sprintf("configuration.income_recipients.donations.%s has to be valid PKH", k))
	}

	for k, v := range configuration.Delegators.Overrides {
		_, err := tezos.ParseAddress(k)
		_assert(err == nil, fmt.Sprintf("configuration.delegators.overrides.%s has to be valid PKH", k))
		_assert(utils.IsPortionWithin0n1(v.Fee),
			getPortionRangeError(fmt.Sprintf("configuration.delegators.overrides.%s fee", k), v.Fee))
	}

	for _, v := range configuration.NotificationConfigurations {
		if !v.IsValid {
			continue
		}
		err := notifications.ValidateNotificatorConfiguration(v.Type, v.Configuration)
		_assert(err == nil, fmt.Sprintf("configuration.notifications.%s has invalid configuration - %s", v.Type, err.Error()))
	}

	return
}
