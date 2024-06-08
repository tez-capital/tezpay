package generate

import (
	"log/slog"

	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

type PayoutSimulatedCandidateValidation func(candidate *PayoutCandidateSimulated, configuration *configuration.RuntimeConfiguration, overrides *configuration.RuntimeDelegatorOverride)
type PayoutSimulatedCandidateValidator struct {
	Id       string
	Validate PayoutSimulatedCandidateValidation
}

type PayoutSimulatedValidationContext struct {
	Configuration *configuration.RuntimeConfiguration
	Overrides     *configuration.RuntimeDelegatorOverride
	Payout        *PayoutCandidateSimulated
}

func (validationContext *PayoutSimulatedValidationContext) ToPayoutCandidateSimulated() *PayoutCandidateSimulated {
	return validationContext.Payout
}

func (validationContext *PayoutSimulatedValidationContext) Validate(validators ...PayoutSimulatedCandidateValidator) *PayoutSimulatedValidationContext {
	if validationContext.Payout.IsInvalid || len(validators) == 0 {
		return validationContext
	}
	for _, validator := range validators {
		slog.Debug("validating payout", "recipient", validationContext.Payout.Recipient, "validator", validator.Id)
		validator.Validate(validationContext.Payout, validationContext.Configuration, validationContext.Overrides)
		slog.Debug("payout validation result", "recipient", validationContext.Payout.Recipient, "is_valid", !validationContext.Payout.IsInvalid)
		if validationContext.Payout.IsInvalid {
			break
		}
	}
	return validationContext
}

// validation

func ValidateSimulatedMinumumAmount(candidate *PayoutCandidateSimulated, configuration *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride) {
	treshhold := configuration.PayoutConfiguration.MinimumAmount
	if treshhold.IsNeg() || candidate.TxKind != enums.PAYOUT_TX_KIND_TEZ { // if payout is not tezos we respect anything above 0
		treshhold = tezos.Zero
	}
	diff := candidate.BondsAmount.Sub(treshhold)
	if diff.IsNeg() || diff.IsZero() {
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_PAYOUT_BELLOW_MINIMUM
	}
}

// Validators
var (
	MinumumAmountSimulatedValidator = PayoutSimulatedCandidateValidator{Id: "MinumumAmountValidator", Validate: ValidateSimulatedMinumumAmount}
)
