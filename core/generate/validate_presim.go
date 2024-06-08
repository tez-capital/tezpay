package generate

import (
	"log/slog"

	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants/enums"
)

type PresimPayoutCandidate = PayoutCandidateWithBondAmountAndFee

type PresimPayoutCandidateValidation func(candidate *PresimPayoutCandidate, configuration *configuration.RuntimeConfiguration)
type PresimPayoutCandidateValidator struct {
	Id       string
	Validate PresimPayoutCandidateValidation
}

type PresimPayoutCandidateValidationContext struct {
	Configuration *configuration.RuntimeConfiguration
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
		validator.Validate(validationContext.Payout, validationContext.Configuration)
		slog.Debug("payout validation result", "recipient", validationContext.Payout.Recipient, "is_valid", !validationContext.Payout.IsInvalid)
		if validationContext.Payout.IsInvalid {
			break
		}
	}
	return validationContext
}

// validation
func ValidateTxKind(candidate *PresimPayoutCandidate, _ *configuration.RuntimeConfiguration) {
	switch candidate.TxKind {
	case enums.PAYOUT_TX_KIND_FA1_2:
	case enums.PAYOUT_TX_KIND_FA2:
	case enums.PAYOUT_TX_KIND_TEZ:
	default:
		candidate.IsInvalid = true
		candidate.InvalidBecause = enums.INVALID_UNSUPPORTED_TX_KIND
	}
}

// Validators
var (
	TxKindValidator = PresimPayoutCandidateValidator{Id: "TxKindValidator", Validate: ValidateTxKind}
)
