package stages

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/core/common"
)

type PayoutCandidate struct {
	Source         tezos.Address
	Recipient      tezos.Address
	FeeRate        float64
	Balance        tezos.Z
	IsInvalid      bool
	IsEmptied      bool
	InvalidBecause enums.EPayoutInvalidReason
}

func (candidate *PayoutCandidate) ToValidationContext(ctx *Context) PayoutValidationContext {
	pkh, _ := candidate.Recipient.MarshalText()
	var overrides *configuration.RuntimeDelegatorOverride
	if delegatorOverride, found := ctx.configuration.Delegators.Overrides[string(pkh)]; found {
		overrides = &delegatorOverride
	}
	return PayoutValidationContext{
		Configuration: ctx.configuration,
		Overrides:     overrides,
		Payout:        candidate,
		Ctx:           ctx,
	}
}

type PayoutCandidateWithBondAmount struct {
	PayoutCandidate
	BondsAmount tezos.Z
}

type PayoutCandidateWithBondAmountAndFee struct {
	PayoutCandidateWithBondAmount
	Fee tezos.Z
}

type PayoutCandidateSimulated struct {
	PayoutCandidateWithBondAmountAndFee
	AllocationBurn int64
	StorageBurn    int64
	OpLimits       *common.OpLimits
}

func (payout *PayoutCandidateSimulated) GetOperationTotalFees() int64 {
	return payout.OpLimits.TransactionFee + payout.AllocationBurn + payout.StorageBurn
}

func (payout *PayoutCandidateSimulated) GetAllocationFee() int64 {
	return payout.AllocationBurn
}

func (payout *PayoutCandidateSimulated) GetOperationFeesWithoutAllocation() int64 {
	return payout.OpLimits.TransactionFee + payout.StorageBurn
}

func (candidate *PayoutCandidateSimulated) ToValidationContext(config *configuration.RuntimeConfiguration) PayoutSimulatedValidationContext {
	pkh, _ := candidate.Recipient.MarshalText()
	var overrides *configuration.RuntimeDelegatorOverride
	if delegatorOverride, found := config.Delegators.Overrides[string(pkh)]; found {
		overrides = &delegatorOverride
	}
	return PayoutSimulatedValidationContext{
		Configuration: config,
		Overrides:     overrides,
		Payout:        candidate,
	}
}

func (payout *PayoutCandidateSimulated) ToPayoutRecipe(baker tezos.Address, cycle int64, kind enums.EPayoutKind) common.PayoutRecipe {
	note := ""
	if payout.IsInvalid {
		kind = enums.PAYOUT_KIND_INVALID
		note = string(payout.InvalidBecause)
	}

	return common.PayoutRecipe{
		Baker:            baker,
		Cycle:            cycle,
		Kind:             kind,
		Delegator:        payout.Source,
		Recipient:        payout.Recipient,
		DelegatedBalance: payout.Balance,
		Amount:           payout.BondsAmount,
		FeeRate:          payout.FeeRate,
		Fee:              payout.Fee,
		OpLimits:         payout.OpLimits,
		Note:             note,
		IsValid:          !payout.IsInvalid,
	}
}

func DelegatorToPayoutCandidate(delegator common.Delegator, configuration *configuration.RuntimeConfiguration) PayoutCandidate {
	pkh, _ := delegator.Address.MarshalText()
	delegatorOverrides := configuration.Delegators.Overrides
	payoutFeeRate := configuration.PayoutConfiguration.Fee
	payoutRecipient := delegator.Address
	if delegatorOverride, ok := delegatorOverrides[string(pkh)]; ok {
		if !delegatorOverride.Recipient.Equal(tezos.InvalidAddress) {
			payoutRecipient = delegatorOverride.Recipient
		}
		if delegatorOverride.Fee != 0 {
			payoutFeeRate = delegatorOverride.Fee
		}
		if delegatorOverride.NoFee {
			payoutFeeRate = 0.
		}
	}
	return PayoutCandidate{
		Source:    delegator.Address,
		Recipient: payoutRecipient,
		FeeRate:   payoutFeeRate,
		Balance:   delegator.Balance,
		IsEmptied: delegator.Emptied,
	}
}
