package prepare

import (
	"log/slog"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

type PayoutRecipeValidationContext struct {
	Configuration *configuration.RuntimeConfiguration
	Overrides     *configuration.RuntimeDelegatorOverride
	Payout        *common.AccumulatedPayoutRecipe
}

func createValidationContext(candidate *common.AccumulatedPayoutRecipe, config *configuration.RuntimeConfiguration) PayoutRecipeValidationContext {
	pkh, _ := candidate.Recipient.MarshalText()
	var overrides *configuration.RuntimeDelegatorOverride
	if delegatorOverride, found := config.Delegators.Overrides[string(pkh)]; found {
		overrides = &delegatorOverride
	}
	return PayoutRecipeValidationContext{
		Configuration: config,
		Overrides:     overrides,
		Payout:        candidate,
	}
}

type PayoutRecipeValidation func(candidate *common.AccumulatedPayoutRecipe, configuration *configuration.RuntimeConfiguration, overrides *configuration.RuntimeDelegatorOverride)
type PayoutRecipeValidator struct {
	Id       string
	Validate PayoutRecipeValidation
}

func (validationContext *PayoutRecipeValidationContext) Unwrap() *common.AccumulatedPayoutRecipe {
	return validationContext.Payout
}

func (validationContext *PayoutRecipeValidationContext) Validate(validators ...PayoutRecipeValidator) *PayoutRecipeValidationContext {
	if !validationContext.Payout.IsValid || len(validators) == 0 {
		return validationContext
	}
	for _, validator := range validators {
		slog.Debug("validating payout", "recipient", validationContext.Payout.Recipient, "validator", validator.Id)
		validator.Validate(validationContext.Payout, validationContext.Configuration, validationContext.Overrides)
		slog.Debug("payout validation result", "recipient", validationContext.Payout.Recipient, "is_valid", validationContext.Payout.IsValid)
		if !validationContext.Payout.IsValid {
			break
		}
	}
	return validationContext
}

// validation

func ValidateMinumumAmount(candidate *common.AccumulatedPayoutRecipe, configuration *configuration.RuntimeConfiguration, _ *configuration.RuntimeDelegatorOverride) {
	treshhold := configuration.PayoutConfiguration.MinimumAmount
	if treshhold.IsNeg() || candidate.TxKind != enums.PAYOUT_TX_KIND_TEZ { // if payout is not tezos we respect anything above 0
		treshhold = tezos.Zero
	}
	diff := candidate.GetAmount().Sub(treshhold)
	if diff.IsNeg() || diff.IsZero() {
		candidate.IsValid = false
		candidate.Note = string(enums.INVALID_PAYOUT_BELLOW_MINIMUM)
	}
}

// Validators
var (
	MinumumAmountValidator = PayoutRecipeValidator{Id: "MinumumAmountValidator", Validate: ValidateMinumumAmount}
)
