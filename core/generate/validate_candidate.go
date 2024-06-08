package generate

import (
	"log/slog"
	"strings"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/state"
	"github.com/trilitech/tzgo/tezos"
)

type PayoutCandidateValidation func(candidate *PayoutCandidate, configuration *configuration.RuntimeConfiguration, overrides *configuration.RuntimeDelegatorOverride, ctx *PayoutGenerationContext)
type PayoutCandidateValidator struct {
	Id       string
	Validate PayoutCandidateValidation
}
type PayoutValidationContext struct {
	Configuration *configuration.RuntimeConfiguration
	Overrides     *configuration.RuntimeDelegatorOverride
	Payout        *PayoutCandidate
	Ctx           *PayoutGenerationContext
}

func (validationContext *PayoutValidationContext) ToPayoutCandidate() *PayoutCandidate {
	return validationContext.Payout
}

func (validationContext *PayoutValidationContext) Validate(validators ...PayoutCandidateValidator) *PayoutValidationContext {
	if validationContext.Payout.IsInvalid || len(validators) == 0 {
		return validationContext
	}
	for _, validator := range validators {
		slog.Debug("validating payout", "recipient", validationContext.Payout.Recipient, "validator", validator.Id)
		validator.Validate(validationContext.Payout, validationContext.Configuration, validationContext.Overrides, validationContext.Ctx)
		slog.Debug("payout validation result", "recipient", validationContext.Payout.Recipient, "is_valid", !validationContext.Payout.IsInvalid)
		if validationContext.Payout.IsInvalid {
			break
		}
	}
	return validationContext
}

// validations
func ValidateRecipient(candidate *PayoutCandidate, _ *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride, _ *PayoutGenerationContext) {
	if candidate.Recipient.Equal(tezos.InvalidAddress) {
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_INVALID_ADDRESS
	}
}

func ValidateMinimumBalance(candidate *PayoutCandidate, configuration *configuration.RuntimeConfiguration, overrides *configuration.RuntimeDelegatorOverride, _ *PayoutGenerationContext) {
	treshhold := configuration.Delegators.Requirements.MinimumBalance
	if overrides != nil {
		if !overrides.MinimumBalance.IsZero() && !overrides.MinimumBalance.IsNeg() {
			treshhold = overrides.MinimumBalance
		}
	}

	if candidate.GetEffectiveBalance().Sub(treshhold).IsNeg() || candidate.GetEffectiveBalance().Sub(treshhold).IsZero() {
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_DELEGATOR_LOW_BAlANCE
	}
}

func ValidateEmptied(candidate *PayoutCandidate, configuration *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride, _ *PayoutGenerationContext) {
	if configuration.PayoutConfiguration.IgnoreEmptyAccounts && candidate.IsEmptied {
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_DELEGATOR_EMPTIED
	}
}

func ValidateIsPrefiltered(candidate *PayoutCandidate, configuration *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride, _ *PayoutGenerationContext) {
	if len(configuration.Delegators.Prefilter) > 0 && !lo.ContainsBy(configuration.Delegators.Prefilter, func(addr tezos.Address) bool { return addr.Equal(candidate.Source) }) {
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_DELEGATOR_PREFILTERED
	}
}

func ValidateIsIgnored(candidate *PayoutCandidate, configuration *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride, _ *PayoutGenerationContext) {
	if lo.ContainsBy(configuration.Delegators.Ignore, func(addr tezos.Address) bool { return addr.Equal(candidate.Source) }) {
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_DELEGATOR_IGNORED
	}
}

func ValidateIgnoreKt(candidate *PayoutCandidate, configuration *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride, _ *PayoutGenerationContext) {
	if configuration.Network.DoNotPaySmartContracts && candidate.Recipient.IsContract() {
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_KT_IGNORED
	}
}

func ValidateRecipientNotBaker(candidate *PayoutCandidate, configuration *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride, ctx *PayoutGenerationContext) {
	if ctx.PayoutKey.Address().Equal(candidate.Recipient) {
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_RECIPIENT_TARGETS_PAYOUT
	}
}

func ValidateNotExcludedPrefix(candidate *PayoutCandidate, configuration *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride, ctx *PayoutGenerationContext) {
	if !strings.HasPrefix(candidate.Recipient.String(), state.Global.GetPayOnlyAddressPrefix()) {
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_MANUALLY_EXCLUDED_BY_PREFIX
	}
}

// Validators
var (
	RecipientValidator         = PayoutCandidateValidator{Id: "RecipientValidator", Validate: ValidateRecipient}
	MinimumBalanceValidator    = PayoutCandidateValidator{Id: "MinimumBalanceValidator", Validate: ValidateMinimumBalance}
	Emptiedalidator            = PayoutCandidateValidator{Id: "Emptiedalidator", Validate: ValidateEmptied}
	IsIgnoredValidator         = PayoutCandidateValidator{Id: "IsIgnoredValidator", Validate: ValidateIsIgnored}
	IsPrefilteredValidator     = PayoutCandidateValidator{Id: "IsPrefilteredValidator", Validate: ValidateIsPrefiltered}
	IgnoreKtValidator          = PayoutCandidateValidator{Id: "IgnoreKtValidator", Validate: ValidateIgnoreKt}
	RecipientNotBaker          = PayoutCandidateValidator{Id: "RecipientNotBaker", Validate: ValidateRecipientNotBaker}
	NotExcludedByAddressPrefix = PayoutCandidateValidator{Id: "NotExcludedByAddressPrefix", Validate: ValidateNotExcludedPrefix}
)
