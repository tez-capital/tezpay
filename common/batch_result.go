package common

import (
	"github.com/samber/lo"
	"github.com/trilitech/tzgo/tezos"
)

type BatchResult struct {
	Payouts   []PayoutRecipe `json:"payouts"`
	OpHash    tezos.OpHash   `json:"op_hash"`
	IsSuccess bool           `json:"is_success"`
	Err       error          `json:"err"`
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
		payout.Note = note

		result[i] = payout.ToPayoutReport()
		result[i].OpHash = br.OpHash
		result[i].IsSuccess = br.IsSuccess
	}
	return result
}

type BatchResults []BatchResult

func (brs BatchResults) ToReports() []PayoutReport {
	return lo.Flatten(lo.Map(brs, func(br BatchResult, _ int) []PayoutReport { return br.ToReports() }))
}
