package common

import (
	"time"

	"blockwatch.cc/tzgo/tezos"
)

type BatchResult struct {
	Payouts   []PayoutRecipe
	OpHash    tezos.OpHash
	IsSuccess bool
	Err       error
}

func NewFailedBatchResult(payouts []PayoutRecipe, err error) *BatchResult {
	return &BatchResult{
		Payouts:   payouts,
		Err:       err,
		IsSuccess: false,
	}
}

func NewFailedBatchResultWithOpHash(Payouts []PayoutRecipe, opHash tezos.OpHash, err error) *BatchResult {
	result := NewFailedBatchResult(Payouts, err)
	result.OpHash = opHash
	return result
}

func NewSuccessBatchResult(payouts []PayoutRecipe, opHash tezos.OpHash) *BatchResult {
	return &BatchResult{
		Payouts:   payouts,
		OpHash:    opHash,
		IsSuccess: true,
	}
}

func (br *BatchResult) ToReports() []PayoutReport {
	result := make([]PayoutReport, len(br.Payouts))
	for i, payout := range br.Payouts {
		note := payout.Note
		if !br.IsSuccess {
			note = br.Err.Error()
		}

		result[i] = PayoutReport{
			Baker:            payout.Baker,
			Timestamp:        time.Now(),
			Cycle:            payout.Cycle,
			Kind:             payout.Kind,
			Delegator:        payout.Delegator,
			DelegatedBalance: payout.DelegatedBalance,
			Recipient:        payout.Recipient,
			Amount:           payout.Amount,
			FeeRate:          payout.FeeRate,
			Fee:              payout.Fee,
			TransactionFee:   payout.OpLimits.TransactionFee,
			OpHash:           br.OpHash,
			IsSuccess:        br.IsSuccess,
			Note:             note,
		}
	}
	return result
}
