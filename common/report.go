package common

import (
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants/enums"
)

type PayoutReport struct {
	Baker            tezos.Address                `json:"baker" csv:"baker"`
	Timestamp        time.Time                    `json:"timestamp" csv:"timestamp"`
	Cycle            int64                        `json:"cycle" csv:"cycle"`
	Kind             enums.EPayoutKind            `json:"kind,omitempty" csv:"kind"`
	TxKind           enums.EPayoutTransactionKind `json:"tx_kind,omitempty" csv:"op_kind"`
	Contract         tezos.Address                `json:"contract,omitempty" csv:"contract"`
	Delegator        tezos.Address                `json:"delegator,omitempty" csv:"delegator"`
	DelegatedBalance tezos.Z                      `json:"delegator_balance,omitempty" csv:"delegator_balance"`
	Recipient        tezos.Address                `json:"recipient,omitempty" csv:"recipient"`
	Amount           tezos.Z                      `json:"amount,omitempty" csv:"amount"`
	FeeRate          float64                      `json:"fee_rate,omitempty" csv:"fee_rate"`
	Fee              tezos.Z                      `json:"fee,omitempty" csv:"fee"`
	TransactionFee   int64                        `json:"tx_fee,omitempty" csv:"tx_fee"`
	OpHash           tezos.OpHash                 `json:"op_hash,omitempty" csv:"op_hash"`
	IsSuccess        bool                         `json:"success" csv:"success"`
	Note             string                       `json:"note,omitempty" csv:"note"`
}

type PayoutCycleReport struct {
	Cycle   int64               `json:"cycle"`
	Invalid []PayoutRecipe      `json:"invalid,omitempty"`
	Payouts []PayoutReport      `json:"payouts"`
	Sumary  *CyclePayoutSummary `json:"summary"`
}
