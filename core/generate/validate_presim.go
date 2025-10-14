package generate

import (
	"log/slog"

	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

type PresimPayoutCandidate = PayoutCandidateWithBondAmountAndFee

type PresimPayoutCandidateValidation func(candidate *PresimPayoutCandidate, configuration *configuration.RuntimeConfiguration, overrides *configuration.RuntimeDelegatorOverride)
type PresimPayoutCandidateValidator struct {
	Id       string
	Validate PresimPayoutCandidateValidation
}

type PresimPayoutCandidateValidationContext struct {
	Configuration *configuration.RuntimeConfiguration
	Overrides     *configuration.RuntimeDelegatorOverride
	Payout        *PresimPayoutCandidate
}

func (validationContext *PresimPayoutCandidateValidationContext) ToPresimPayoutCandidate() *PresimPayoutCandidate {
	return validationContext.Payout
}

func (validationContext *PresimPayoutCandidateValidationContext) Validate(validators ...PresimPayoutCandidateValidator) *PresimPayoutCandidateValidationContext {
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
func ValidateTxKind(candidate *PresimPayoutCandidate, _ *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride) {
	switch candidate.TxKind {
	case enums.PAYOUT_TX_KIND_FA1_2:
	case enums.PAYOUT_TX_KIND_FA2:
	case enums.PAYOUT_TX_KIND_TEZ:
	default:
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_UNSUPPORTED_TX_KIND
	}
}

func ValidateMinumumAmount(candidate *PayoutCandidateWithBondAmountAndFee, configuration *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride) {
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
	TxKindValidator        = PresimPayoutCandidateValidator{Id: "TxKindValidator", Validate: ValidateTxKind}
	MinumumAmountValidator = PresimPayoutCandidateValidator{Id: "MinumumAmountValidator", Validate: ValidateMinumumAmount}
)
