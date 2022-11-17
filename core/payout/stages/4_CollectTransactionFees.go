package stages

import (
	"blockwatch.cc/tzgo/codec"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

const (
	TX_BATCH_CAPACITY = 20
)

func batchEstimate(payouts []PayoutCandidateWithBondAmountAndFee, ctx Context) []PayoutCandidateSimulated {
	candidates := lo.Filter(payouts, func(payout PayoutCandidateWithBondAmountAndFee, _ int) bool {
		return !payout.IsInvalid
	})
	invalid := lo.Map(lo.Filter(payouts, func(payout PayoutCandidateWithBondAmountAndFee, _ int) bool {
		return payout.IsInvalid
	}), func(candidate PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateSimulated {
		return PayoutCandidateSimulated{
			PayoutCandidateWithBondAmountAndFee: candidate,
		}
	})
	batches := make([][]PayoutCandidateWithBondAmountAndFee, 0)
	for offset := 0; offset < len(candidates); offset += TX_BATCH_CAPACITY {
		batches = append(batches, lo.Slice(candidates, offset, offset+TX_BATCH_CAPACITY))
	}
	batchesSimulated := lo.Map(batches, func(batch []PayoutCandidateWithBondAmountAndFee, index int) []PayoutCandidateSimulated {
		op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
		for _, p := range batch {
			op.WithTransfer(p.Recipient, p.BondsAmount.Int64())
		}
		receipt, err := ctx.Collector.Simulate(op, ctx.PayoutKey)
		if err != nil || !receipt.IsSuccess() {
			log.Tracef("failed to estimate tx costs of batch n.%d (falling back to one by one estimate)", index)
			return lo.Map(batch, func(candidate PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateSimulated {
				op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
				op.WithTransfer(candidate.Recipient, candidate.BondsAmount.Int64())

				receipt, err := ctx.Collector.Simulate(op, ctx.PayoutKey)
				if err != nil || !receipt.IsSuccess() {
					log.Warnf("failed to estimate tx costs to '%s' (delegator: '%s', amount %d)", candidate.Recipient, candidate.Source, candidate.BondsAmount.Int64())
					if err != nil {
						log.Debugf(err.Error())
					} else {
						log.Debugf(receipt.Error().Error())
					}
					candidate.IsInvalid = true
					candidate.InvalidBecause = enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS
					return PayoutCandidateSimulated{
						PayoutCandidateWithBondAmountAndFee: candidate,
					}
				}

				costs := receipt.TotalCosts()

				return PayoutCandidateSimulated{
					PayoutCandidateWithBondAmountAndFee: candidate,
					AllocationBurn:                      costs.AllocationBurn,
					StorageBurn:                         costs.StorageBurn,
					OpLimits: &common.OpLimits{
						GasLimit:       costs.GasUsed + constants.GAS_LIMIT_BUFFER,
						StorageLimit:   utils.CalculateStorageLimit(costs),
						TransactionFee: utils.EstimateTransactionFee(op, receipt.Costs()),
					},
				}
			})
		}
		costs := receipt.Costs()
		return lo.Map(batch, func(candidate PayoutCandidateWithBondAmountAndFee, index int) PayoutCandidateSimulated {
			if index >= len(costs) {
				panic("Partial estimate. This should never happen!")
			}
			return PayoutCandidateSimulated{
				PayoutCandidateWithBondAmountAndFee: candidate,
				AllocationBurn:                      costs[index].AllocationBurn,
				StorageBurn:                         costs[index].StorageBurn,
				OpLimits: &common.OpLimits{
					GasLimit:       costs[index].GasUsed + constants.GAS_LIMIT_BUFFER,
					StorageLimit:   utils.CalculateStorageLimit(costs[index]),
					TransactionFee: utils.EstimateContentFee(op.Contents[index], costs[index], op.Params, true),
				},
			}
		})
	})

	return append(invalid, lo.Flatten(batchesSimulated)...)
}

func collectTransactionFees(ctx Context) (result Context, err error) {
	configuration := ctx.GetConfiguration()
	candidates := ctx.StageData.PayoutCandidatesWithBondAmountAndFees

	log.Debug("simulating transactions to collect tx fees")
	simulatedPayouts := batchEstimate(candidates, ctx)

	simulatedPayouts = lo.Map(simulatedPayouts, func(candidate PayoutCandidateSimulated, _ int) PayoutCandidateSimulated {
		if candidate.IsInvalid {
			return candidate
		}
		if !configuration.PayoutConfiguration.IsPayingTxFee {
			candidate.BondsAmount = candidate.BondsAmount.Sub64(candidate.GetOperationFeesWithoutAllocation())
		}
		if !configuration.PayoutConfiguration.IsPayingAllocationTxFee {
			candidate.BondsAmount = candidate.BondsAmount.Sub64(candidate.GetAllocationFee())
		}
		return candidate
	})
	ctx.StageData.PayoutCandidatesSimulated = simulatedPayouts
	return ctx, nil
}

var CollectTransactionFees = WrapStage(collectTransactionFees)
