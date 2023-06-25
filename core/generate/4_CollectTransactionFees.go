package generate

import (
	"errors"
	"fmt"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

const (
	TX_BATCH_CAPACITY = 20
)

func buildOpForEstimation(ctx *PayoutGenerationContext, batch []PayoutCandidateWithBondAmountAndFee, doubleFirstTx bool) (*codec.Op, error) {
	var err error
	op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
	op.WithTTL(constants.MAX_OPERATION_TTL)
	for _, p := range batch {
		if err = common.InjectTransferContents(op, ctx.PayoutKey.Address(), &p); err != nil {
			break
		}
	}
	if doubleFirstTx && err == nil {
		err = common.InjectTransferContents(op, ctx.PayoutKey.Address(), &batch[0])
	}
	return op, err
}

func estimateBatchFees(batch []PayoutCandidateWithBondAmountAndFee, ctx *PayoutGenerationContext) ([]PayoutCandidateSimulationResult, error) {
	var (
		err     error
		receipt *rpc.Receipt
	)
	op, err := buildOpForEstimation(ctx, batch, true)

	if err != nil {
		return nil, err
	}
	receipt, err = ctx.GetCollector().Simulate(op, ctx.PayoutKey)
	if err != nil || (receipt != nil && !receipt.IsSuccess()) {
		if receipt != nil && receipt.Error() != nil && (err == nil || receipt.Error().Error() != err.Error()) {
			return nil, errors.Join(receipt.Error(), err)
		}
		return nil, err
	}

	costs := receipt.Op.Costs()
	if len(costs) < 2 {
		panic("Partial estimate. This should never happen!")
	}

	serializationFee := costs[0].GasUsed - costs[len(costs)-1].GasUsed

	costs[0].GasUsed = costs[len(costs)-1].GasUsed // we replace with actual costs - without deserialization of everything
	costs = costs[:len(costs)-1]
	result := make([]PayoutCandidateSimulationResult, 0)

	// remove last op content as it is stored in first alreedy
	op.Contents = op.Contents[:len(op.Contents)-1]
	totalBytes := 0
	// we intentionally caclculate total bytes on contents without last one (the one used to determinie serialization fee)
	// to slightly increase per tx serialization fee as a buffer, it is likely going to be eaten be division anyway
	for _, v := range op.Contents {
		bytes, err := v.MarshalBinary()
		if err != nil {
			return nil, err
		}
		totalBytes += len(bytes)
	}

	for i, p := range costs {
		bytes, err := op.Contents[i].MarshalBinary()
		if err != nil {
			return nil, err
		}
		//rebuild op for estimates
		op, err := buildOpForEstimation(ctx, []PayoutCandidateWithBondAmountAndFee{batch[i]}, false)
		if err != nil {
			return nil, err
		}

		txSerializationFee := (serializationFee * int64(len(bytes))) / int64(totalBytes)
		result = append(result, PayoutCandidateSimulationResult{
			AllocationBurn: p.AllocationBurn,
			StorageBurn:    p.StorageBurn,
			OpLimits: &common.OpLimits{
				GasLimit:         p.GasUsed + ctx.configuration.PayoutConfiguration.TxGasLimitBuffer,
				StorageLimit:     utils.CalculateStorageLimit(p),
				TransactionFee:   utils.EstimateTransactionFee(op, costs, serializationFee+ctx.configuration.PayoutConfiguration.TxDeserializationGasBuffer+ctx.configuration.PayoutConfiguration.TxGasLimitBuffer),
				SerializationFee: txSerializationFee + ctx.configuration.PayoutConfiguration.TxDeserializationGasBuffer,
			},
		})
	}

	return result, err
}

// returns simulated payouts and serialization costs
func estimateTransactionFees(payouts []PayoutCandidateWithBondAmountAndFee, ctx *PayoutGenerationContext) []PayoutCandidateSimulated {
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
		simulationResults, err := estimateBatchFees(batch, ctx)
		if err != nil {
			log.Tracef("failed to estimate tx costs of batch n.%d (falling back to one by one estimate)", index)
			return lo.Map(batch, func(candidate PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateSimulated {
				simulationResults, err := estimateBatchFees([]PayoutCandidateWithBondAmountAndFee{candidate}, ctx)
				if len(simulationResults) == 0 {
					err = fmt.Errorf("unexpected simulation results: %v", simulationResults)
				}
				if err != nil {
					log.Warnf("failed to estimate tx costs to '%s' (delegator: '%s', amount %d, kind '%s')", candidate.Recipient, candidate.Source, candidate.BondsAmount.Int64(), candidate.TxKind)
					candidate.IsInvalid = true
					candidate.InvalidBecause = enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS
					return PayoutCandidateSimulated{
						PayoutCandidateWithBondAmountAndFee: candidate,
					}
				}

				return PayoutCandidateSimulated{
					PayoutCandidateWithBondAmountAndFee: candidate,
					PayoutCandidateSimulationResult:     simulationResults[0],
				}
			})
		}
		return lo.Map(batch, func(candidate PayoutCandidateWithBondAmountAndFee, index int) PayoutCandidateSimulated {
			if index >= len(simulationResults) {
				panic("Partial estimate. This should never happen!")
			}
			return PayoutCandidateSimulated{
				PayoutCandidateWithBondAmountAndFee: candidate,
				PayoutCandidateSimulationResult:     simulationResults[index],
			}
		})
	})

	return append(invalid, lo.Flatten(batchesSimulated)...)
}

func CollectTransactionFees(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (result *PayoutGenerationContext, err error) {
	candidates := ctx.StageData.PayoutCandidatesWithBondAmountAndFees

	// presim validation
	candidates = lo.Map(candidates, func(candidate PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateWithBondAmountAndFee {
		validationContext := candidate.ToValidationContext(ctx)
		return *validationContext.Validate(
			TxKindValidator,
		).ToPresimPayoutCandidate()
	})

	log.Debug("simulating transactions to collect tx fees")
	simulatedPayouts := estimateTransactionFees(candidates, ctx)

	simulatedPayouts = lo.Map(simulatedPayouts, func(candidate PayoutCandidateSimulated, _ int) PayoutCandidateSimulated {
		if candidate.IsInvalid || candidate.TxKind != enums.PAYOUT_TX_KIND_TEZ { // we don't collect fees from non-tez payouts
			return candidate
		}
		if !candidate.IsBakerPayingTxFee {
			candidate.BondsAmount = candidate.BondsAmount.Sub64(candidate.GetOperationFeesWithoutAllocation())
		}
		if !candidate.IsBakerPayingAllocationTxFee {
			candidate.BondsAmount = candidate.BondsAmount.Sub64(candidate.GetAllocationFee())
		}
		return candidate
	})
	ctx.StageData.PayoutCandidatesSimulated = simulatedPayouts
	return ctx, nil
}
