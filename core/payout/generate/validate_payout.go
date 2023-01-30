package generate

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
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
		log.Tracef("validating payout to %s with %s", validationContext.Payout.Recipient, validator.Id)
		validator.Validate(validationContext.Payout, validationContext.Configuration, validationContext.Overrides, validationContext.Ctx)
		log.Tracef("payout to %s validation result: %t", validationContext.Payout.Recipient, !validationContext.Payout.IsInvalid)
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

	if candidate.Balance.Sub(treshhold).IsNeg() || candidate.Balance.Sub(treshhold).IsZero() {
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

// Validators
var (
	RecipientValidator      = PayoutCandidateValidator{Id: "RecipientValidator", Validate: ValidateRecipient}
	MinimumBalanceValidator = PayoutCandidateValidator{Id: "MinimumBalanceValidator", Validate: ValidateMinimumBalance}
	Emptiedalidator         = PayoutCandidateValidator{Id: "Emptiedalidator", Validate: ValidateEmptied}
	IsIgnoredValidator      = PayoutCandidateValidator{Id: "IsIgnoredValidator", Validate: ValidateIsIgnored}
	IgnoreKtValidator       = PayoutCandidateValidator{Id: "IgnoreKtValidator", Validate: ValidateIgnoreKt}
	RecipientNotBaker       = PayoutCandidateValidator{Id: "RecipientNotBaker", Validate: ValidateRecipientNotBaker}
)
