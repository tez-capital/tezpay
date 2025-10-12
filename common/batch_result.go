package common

import (
	"github.com/samber/lo"
	"github.com/trilitech/tzgo/tezos"
)

type BatchResult struct {
	Payouts   []*AccumulatedPayoutRecipe `json:"payouts"`
	OpHash    tezos.OpHash               `json:"op_hash"`
	IsSuccess bool                       `json:"is_success"`
	Err       error                      `json:"err"`
}

func NewFailedBatchResult(payouts []*AccumulatedPayoutRecipe, err error) *BatchResult {
	return &BatchResult{
		Payouts:   payouts,
		Err:       err,
		IsSuccess: false,
	}
}

func NewFailedBatchResultWithOpHash(Payouts []*AccumulatedPayoutRecipe, opHash tezos.OpHash, err error) *BatchResult {
	result := NewFailedBatchResult(Payouts, err)
	result.OpHash = opHash
	return result
}

func NewSuccessBatchResult(payouts []*AccumulatedPayoutRecipe, opHash tezos.OpHash) *BatchResult {
	return &BatchResult{
		Payouts:   payouts,
		OpHash:    opHash,
		IsSuccess: true,
	}
}

func (br *BatchResult) ToIndividualReports() []PayoutReport {
	result := make([]PayoutReport, 0, len(br.Payouts))
	for _, payout := range br.Payouts {
		for _, acc := range payout.Accumulated {
			note := acc.Note
			if !br.IsSuccess {
				note = br.Err.Error()
			}
			report := acc.ToPayoutReport()
			report.OpHash = br.OpHash
			report.IsSuccess = br.IsSuccess
			report.Note = note
			result = append(result, report)
		}
	}
	return result
}

type BatchResults []BatchResult

func (brs BatchResults) ToIndividualReports() []PayoutReport {
	return lo.Flatten(lo.Map(brs, func(br BatchResult, _ int) []PayoutReport { return br.ToIndividualReports() }))
}
