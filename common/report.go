package common

import (
	"time"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

type PayoutReport struct {
	Id               string                       `json:"id" csv:"id"`
	Baker            tezos.Address                `json:"baker" csv:"baker"`
	Timestamp        time.Time                    `json:"timestamp" csv:"timestamp"`
	Cycle            int64                        `json:"cycle" csv:"cycle"`
	Kind             enums.EPayoutKind            `json:"kind,omitempty" csv:"kind"`
	TxKind           enums.EPayoutTransactionKind `json:"tx_kind,omitempty" csv:"op_kind"`
	FAContract       tezos.Address                `json:"contract,omitempty" csv:"contract"`
	FATokenId        tezos.Z                      `json:"token_id,omitempty" csv:"token_id"`
	FAAlias          string                       `json:"fa_alias,omitempty" csv:"fa_alias"`
	FADecimals       int                          `json:"fa_decimals,omitempty" csv:"fa_decimals"`
	Delegator        tezos.Address                `json:"delegator,omitempty" csv:"delegator"`
	DelegatedBalance tezos.Z                      `json:"delegator_balance,omitempty" csv:"delegator_balance"`
	StakedBalance    tezos.Z                      `json:"staked_balance,omitempty" csv:"staked_balance"`
	Recipient        tezos.Address                `json:"recipient,omitempty" csv:"recipient"`
	Amount           tezos.Z                      `json:"amount,omitempty" csv:"amount"`
	FeeRate          float64                      `json:"fee_rate,omitempty" csv:"fee_rate"`
	Fee              tezos.Z                      `json:"fee,omitempty" csv:"fee"`
	TransactionFee   int64                        `json:"tx_fee,omitempty" csv:"tx_fee"`
	OpHash           tezos.OpHash                 `json:"op_hash,omitempty" csv:"op_hash"`
	IsSuccess        bool                         `json:"success" csv:"success"`
	Note             string                       `json:"note,omitempty" csv:"note"`

	Accumulated []*PayoutReport `json:"-" csv:"-"` // just for internal linking of accumulated payouts
}

func (pr *PayoutReport) GetTransactionFee() int64 {
	return pr.TransactionFee
}

func (pr *PayoutReport) ToTableRowData() []string {
	return []string{
		ShortenAddress(pr.Delegator),
		ShortenAddress(pr.Recipient),
		MutezToTezS(pr.DelegatedBalance.Int64()),
		string(pr.Kind),
		ShortenAddress(pr.FAContract),
		ToStringEmptyIfZero(pr.FATokenId.Int64()),
		FormatTokenAmount(pr.TxKind, pr.Amount.Int64(), pr.FAAlias, pr.FADecimals),
		FloatToPercentage(pr.FeeRate),
		MutezToTezS(pr.Fee.Int64()),
		MutezToTezS(pr.GetTransactionFee()),
		pr.OpHash.String(),
		pr.Note,
	}
}

func (pr *PayoutReport) Disperse() []PayoutReport {
	if len(pr.Accumulated) == 0 {
		return []PayoutReport{*pr}
	}

	dispersed := make([]PayoutReport, 0, len(pr.Accumulated)+1)
	dispersed = append(dispersed, *pr)

	for _, acc := range pr.Accumulated {
		if acc.Id == pr.Id { // avoid duplicates if first accumulated is the same as parent
			continue
		}
		acc.OpHash = pr.OpHash
		acc.IsSuccess = pr.IsSuccess
		dispersed = append(dispersed, *acc)
	}
	return dispersed
}

func (pr *PayoutReport) GetTableHeaders() []string {
	return []string{
		"Delegator",
		"Recipient",
		"Delegated Balance",
		"Kind",
		"FA Contract",
		"FA Token ID",
		"Amount",
		"Fee Rate",
		"Fee",
		"Transaction Fee",
		"Op Hash",
		"Note",
	}
}

func GetReportsTotals(reports []PayoutReport) []string {
	var totalAmount, totalFee, totalTxFee int64
	for _, report := range reports {
		if report.TxKind == enums.PAYOUT_TX_KIND_TEZ {
			totalAmount += report.Amount.Int64()
		}
		totalFee += report.Fee.Int64()
		totalTxFee += report.GetTransactionFee()
	}
	return []string{
		"",
		"",
		"",
		"",
		"",
		"",
		MutezToTezS(totalAmount),
		"",
		MutezToTezS(totalFee),
		MutezToTezS(totalTxFee),
		"",
		"",
	}
}

// returns total amounts and count of filtered reports
func GetFilteredReportsTotals(reports []PayoutReport, kind enums.EPayoutKind) ([]string, int) {
	r := lo.Filter(reports, func(report PayoutReport, _ int) bool {
		return report.Kind == kind
	})
	return GetReportsTotals(r), len(r)
}

type PayoutCycleReport struct {
	Cycle   int64          `json:"cycle"`
	Invalid []PayoutRecipe `json:"invalid,omitempty"`
	Payouts []PayoutReport `json:"payouts"`
	Sumary  *PayoutSummary `json:"summary"`
}
