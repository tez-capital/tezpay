package common

import (
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants/enums"
)

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
