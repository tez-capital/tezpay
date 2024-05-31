package configuration

import (
	"errors"
	"fmt"

	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/notifications"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	"github.com/trilitech/tzgo/tezos"
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
			err = errors.Join(constants.ErrConfigurationValidationFailed, errors.New(msg))
		}
	}()

	_assert(configuration != nil, "configuration is nil")
	_assert(lo.Contains(enums.SUPPORTED_WALLET_MODES, configuration.PayoutConfiguration.WalletMode),
		fmt.Sprintf("configuration.payouts.wallet_mode - '%s' not supported", configuration.PayoutConfiguration.WalletMode))
	_assert(lo.Contains(enums.SUPPORTED_PAYOUT_MODES, configuration.PayoutConfiguration.PayoutMode),
		fmt.Sprintf("configuration.payouts.payout_mode - '%s' not supported", configuration.PayoutConfiguration.PayoutMode))
	_assert(configuration.PayoutConfiguration.MinimumDelayBlocks <= configuration.PayoutConfiguration.MaximumDelayBlocks,
		"configuration.payouts.minimum_delay_blocks must be less or equal to configuration.payouts.maximum_delay_blocks")

	_assert(lo.Contains(enums.SUPPORTED_DELEGATOR_MINIMUM_BALANCE_REWARD_DESTINATIONS, configuration.Delegators.Requirements.BellowMinimumBalanceRewardDestination),
		fmt.Sprintf("configuration.delegators.requirements.below_minimum_reward_destination - '%s' not supported", configuration.Delegators.Requirements.BellowMinimumBalanceRewardDestination))

	_assert(utils.IsPortionWithin0n1(configuration.PayoutConfiguration.Fee),
		getPortionRangeError("configuration.payouts.fee", configuration.PayoutConfiguration.Fee))
	_assert(utils.IsPortionWithin0n1(configuration.IncomeRecipients.DonateFees),
		getPortionRangeError("configuration.income_recipients.donate/fees", configuration.IncomeRecipients.DonateFees))
	_assert(utils.IsPortionWithin0n1(configuration.IncomeRecipients.DonateBonds),
		getPortionRangeError("configuration.income_recipients.donate/bonds", configuration.IncomeRecipients.DonateBonds))

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
		_assert(v.Fee == nil || utils.IsPortionWithin0n1(*v.Fee),
			getPortionRangeError(fmt.Sprintf("configuration.delegators.overrides.%s fee", k), *v.Fee))
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
