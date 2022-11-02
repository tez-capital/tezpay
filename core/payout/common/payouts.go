package common

import (
	"time"

	"blockwatch.cc/tzgo/tezos"
	tezpay_tezos "github.com/alis-is/tezpay/clients/tezos"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants/enums"
)

type PayoutCandidate struct {
	Source         tezos.Address
	Recipient      tezos.Address
	FeeRate        float32
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
	Candidate   PayoutCandidate
	BondsAmount tezos.Z
}

type PayoutCandidateWithBondAmountAndFee struct {
	Candidate   PayoutCandidate
	BondsAmount tezos.Z
	Fee         tezos.Z
}

type PayoutCandidateSimulated struct {
	Candidate      PayoutCandidate
	BondsAmount    tezos.Z
	Fee            tezos.Z
	AllocationBurn int64
	StorageBurn    int64
	OpLimits       *OpLimits
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
	pkh, _ := candidate.Candidate.Recipient.MarshalText()
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

func (payout *PayoutCandidateSimulated) ToPayoutRecipe(baker tezos.Address, cycle int64, kind enums.EPayoutKind) PayoutRecipe {
	note := ""
	if payout.Candidate.IsInvalid {
		kind = enums.PAYOUT_KIND_INVALID
		note = string(payout.Candidate.InvalidBecause)
	}

	return PayoutRecipe{
		Baker:            baker,
		Cycle:            cycle,
		Kind:             kind,
		Delegator:        payout.Candidate.Source,
		Recipient:        payout.Candidate.Recipient,
		DelegatedBalance: payout.Candidate.Balance,
		Amount:           payout.BondsAmount,
		FeeRate:          payout.Candidate.FeeRate,
		Fee:              payout.Fee,
		OpLimits:         payout.OpLimits,
		Note:             note,
		IsValid:          !payout.Candidate.IsInvalid,
	}
}

func DelegatorToPayoutCandidate(delegator tezpay_tezos.Delegator, configuration *configuration.RuntimeConfiguration) PayoutCandidate {
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

type OpLimits struct {
	TransactionFee int64 `json:"transaction_fee,omitempty"`
	StorageLimit   int64 `json:"storage_limit,omitempty"`
	GasLimit       int64 `json:"gas_limit,omitempty"`
}

type PayoutRecipe struct {
	Baker            tezos.Address     `json:"baker"`
	Delegator        tezos.Address     `json:"delegator,omitempty"`
	Cycle            int64             `json:"cycle,omitempty"`
	Recipient        tezos.Address     `json:"recipient,omitempty"`
	Kind             enums.EPayoutKind `json:"kind,omitempty"`
	DelegatedBalance tezos.Z           `json:"delegator_balance,omitempty"`
	Amount           tezos.Z           `json:"amount,omitempty"`
	FeeRate          float32           `json:"fee_rate,omitempty"`
	Fee              tezos.Z           `json:"fee,omitempty"`
	OpLimits         *OpLimits         `json:"op_limits,omitempty"`
	Note             string            `json:"note,omitempty"`
	IsValid          bool              `json:"valid,omitempty"`
}

func (pr *PayoutRecipe) PayoutRecipeToPayoutReport() PayoutReport {
	txFee := int64(0)
	if pr.OpLimits != nil {
		txFee = pr.OpLimits.TransactionFee
	}

	return PayoutReport{
		Baker:            pr.Baker,
		Timestamp:        time.Now(),
		Cycle:            pr.Cycle,
		Kind:             pr.Kind,
		Delegator:        pr.Delegator,
		DelegatedBalance: pr.DelegatedBalance,
		Recipient:        pr.Recipient,
		Amount:           pr.Amount,
		FeeRate:          pr.FeeRate,
		Fee:              pr.Fee,
		TransactionFee:   txFee,
		OpHash:           tezos.ZeroOpHash,
		IsSuccess:        false,
		Note:             pr.Note,
	}
}

type PayoutContext struct {
	Payouts           []PayoutRecipe
	InvalidCandidates []PayoutRecipe
}

type CyclePayoutSummary struct {
	Cycle              int64   `json:"cycle"`
	Delegators         int     `json:"delegators"`
	StakingBalance     tezos.Z `json:"staking_balance"`
	EarnedFees         tezos.Z `json:"cycle_fees"`
	EarnedRewards      tezos.Z `json:"cycle_rewards"`
	DistributedRewards tezos.Z `json:"distributed_rewards"`
	BondIncome         tezos.Z `json:"bond_income"`
	FeeIncome          tezos.Z `json:"fee_income"`
	IncomeTotal        tezos.Z `json:"total_income"`
	DonatedBonds       tezos.Z `json:"donated_bonds"`
	DonatedFees        tezos.Z `json:"donated_fees"`
	DonatedTotal       tezos.Z `json:"donated_total"`
}

type CyclePayoutBlueprint struct {
	Cycle   int64          `json:"cycle,omitempty"`
	Payouts []PayoutRecipe `json:"payouts,omitempty"`
	Summary CyclePayoutSummary
}
