package generate

import (
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

type PayoutCandidate struct {
	Source                       tezos.Address              `json:"source,omitempty"`
	Recipient                    tezos.Address              `json:"recipient,omitempty"`
	FeeRate                      float64                    `json:"fee_rate,omitempty"`
	StakingBalance               tezos.Z                    `json:"balance,omitempty"`
	DelegatedBalance             tezos.Z                    `json:"delegated_balance,omitempty"`
	IsInvalid                    bool                       `json:"is_invalid,omitempty"`
	IsEmptied                    bool                       `json:"is_emptied,omitempty"`
	IsBakerPayingTxFee           bool                       `json:"is_baker_paying_tx_fee,omitempty"`
	IsBakerPayingAllocationTxFee bool                       `json:"is_baker_paying_allocation_tx_fee,omitempty"`
	InvalidBecause               enums.EPayoutInvalidReason `json:"invalid_because,omitempty"`
	// mainly for accumulation to be able to check if fee was collected and subtract it from the amount
	TxFeeCollected bool `json:"tx_fee_collected,omitempty"`
	// mainly for accumulation to be able to check if fee was collected and subtract it from the amount
	AllocationFeeCollected bool `json:"allocation_fee_collected,omitempty"`
}

func (candidate *PayoutCandidate) GetEffectiveBalance() tezos.Z {
	return candidate.StakingBalance.Add(candidate.DelegatedBalance)
}

func (candidate *PayoutCandidate) ToValidationContext(ctx *PayoutGenerationContext) PayoutValidationContext {
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
	BondsAmount tezos.Z                      `json:"bonds_amount,omitempty"`
	TxKind      enums.EPayoutTransactionKind `json:"tx_kind,omitempty"`
	FATokenId   tezos.Z                      `json:"fa_token_id,omitempty"` // required only if fa12 or fa2
	FAContract  tezos.Address                `json:"fa_contract"`           // required only if fa12 or fa2
}

func (candidate *PayoutCandidateWithBondAmount) GetDestination() tezos.Address {
	return candidate.Recipient
}

func (candidate *PayoutCandidateWithBondAmount) GetTxKind() enums.EPayoutTransactionKind {
	return candidate.TxKind
}

func (candidate *PayoutCandidateWithBondAmount) GetFATokenId() tezos.Z {
	return candidate.FATokenId
}

func (candidate *PayoutCandidateWithBondAmount) GetFAContract() tezos.Address {
	return candidate.FAContract
}

func (candidate *PayoutCandidateWithBondAmount) GetAmount() tezos.Z {
	return candidate.BondsAmount
}

func (candidate *PayoutCandidateWithBondAmount) GetFeeRate() float64 {
	return candidate.FeeRate
}

type PayoutCandidateWithBondAmountAndFee struct {
	PayoutCandidateWithBondAmount
	Fee tezos.Z `json:"fee,omitempty"`
}

func (candidate *PayoutCandidateWithBondAmountAndFee) ToValidationContext(ctx *PayoutGenerationContext) PresimPayoutCandidateValidationContext {
	return PresimPayoutCandidateValidationContext{
		Configuration: ctx.configuration,
		Payout:        candidate,
	}
}

type PayoutCandidateSimulated struct {
	PayoutCandidateWithBondAmountAndFee
	SimulationResult *common.OpLimits
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
		Baker:                  baker,
		Cycle:                  cycle,
		Kind:                   kind,
		TxKind:                 payout.TxKind,
		Delegator:              payout.Source,
		Recipient:              payout.Recipient,
		DelegatedBalance:       payout.DelegatedBalance,
		StakingBalance:         payout.StakingBalance,
		FATokenId:              payout.FATokenId,
		FAContract:             payout.FAContract,
		Amount:                 payout.BondsAmount,
		FeeRate:                payout.FeeRate,
		Fee:                    payout.Fee,
		OpLimits:               payout.SimulationResult,
		TxFeeCollected:         payout.TxFeeCollected,
		AllocationFeeCollected: payout.AllocationFeeCollected,
		Note:                   note,
		IsValid:                !payout.IsInvalid,
	}
}

func DelegatorToPayoutCandidate(delegator common.Delegator, configuration *configuration.RuntimeConfiguration) PayoutCandidate {
	pkh, _ := delegator.Address.MarshalText()
	delegatorOverrides := configuration.Delegators.Overrides
	payoutFeeRate := configuration.PayoutConfiguration.Fee
	payoutRecipient := delegator.Address
	isBakerPayingTxFee := configuration.PayoutConfiguration.IsPayingTxFee
	IsBakerPayingAllocationTxFee := configuration.PayoutConfiguration.IsPayingAllocationTxFee

	if delegatorOverride, ok := delegatorOverrides[string(pkh)]; ok {
		if !delegatorOverride.Recipient.Equal(tezos.InvalidAddress) {
			payoutRecipient = delegatorOverride.Recipient
		}
		if delegatorOverride.Fee != nil {
			payoutFeeRate = *delegatorOverride.Fee
		}
		if delegatorOverride.IsBakerPayingTxFee != nil {
			isBakerPayingTxFee = *delegatorOverride.IsBakerPayingTxFee
		}
		if delegatorOverride.IsBakerPayingAllocationTxFee != nil {
			IsBakerPayingAllocationTxFee = *delegatorOverride.IsBakerPayingAllocationTxFee
		}
		if delegatorOverride.MaximumBalance != nil && delegatorOverride.MaximumBalance.IsLess(delegator.DelegatedBalance) {
			delegator.DelegatedBalance = *delegatorOverride.MaximumBalance
		}
	}

	return PayoutCandidate{
		Source:                       delegator.Address,
		Recipient:                    payoutRecipient,
		FeeRate:                      payoutFeeRate,
		DelegatedBalance:             delegator.DelegatedBalance,
		StakingBalance:               delegator.StakedBalance,
		IsEmptied:                    delegator.Emptied,
		IsBakerPayingTxFee:           isBakerPayingTxFee,
		IsBakerPayingAllocationTxFee: IsBakerPayingAllocationTxFee,
	}
}
