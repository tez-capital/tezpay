package generate

import (
	"errors"
	"fmt"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

const (
	TX_BATCH_CAPACITY = 40
)

func filterCandidatesByTxKind(payouts []PayoutCandidateWithBondAmountAndFee, kinds []enums.EPayoutTransactionKind) []PayoutCandidateWithBondAmountAndFee {
	return lo.Filter(payouts, func(payout PayoutCandidateWithBondAmountAndFee, _ int) bool {
		return lo.Contains(kinds, payout.TxKind)
	})
}

func rejectCandidatesByTxKind(payouts []PayoutCandidateWithBondAmountAndFee, kinds []enums.EPayoutTransactionKind) []PayoutCandidateWithBondAmountAndFee {
	return lo.Filter(payouts, func(payout PayoutCandidateWithBondAmountAndFee, _ int) bool {
		return !lo.Contains(kinds, payout.TxKind)
	})
}

func splitIntoBatches(candidates []PayoutCandidateWithBondAmountAndFee, capacity int) [][]PayoutCandidateWithBondAmountAndFee {
	batches := make([][]PayoutCandidateWithBondAmountAndFee, 0)
	if capacity == 0 {
		capacity = TX_BATCH_CAPACITY
	}
	for offset := 0; offset < len(candidates); offset += capacity {
		batches = append(batches, lo.Slice(candidates, offset, offset+capacity))
	}

	return batches
}

func buildOpForEstimation(ctx *PayoutGenerationContext, batch []PayoutCandidateWithBondAmountAndFee, injectBurnTransactions bool) (*codec.Op, error) {
	var err error
	op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
	op.WithTTL(constants.MAX_OPERATION_TTL)
	if injectBurnTransactions {
		op.WithTransfer(tezos.BurnAddress, 1)
	}
	for _, p := range batch {
		if err = common.InjectTransferContents(op, ctx.PayoutKey.Address(), &p); err != nil {
			break
		}
	}
	if injectBurnTransactions {
		op.WithTransfer(tezos.BurnAddress, 1)
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
	if len(costs) < 3 {
		panic("Partial estimate. This should never happen!")
	}

	serializationGas := costs[0].GasUsed - costs[len(costs)-1].GasUsed
	// remove first and last contents and limits it is only burn tx to measure serialization cost
	costs = costs[1 : len(costs)-1]
	op.Contents = op.Contents[1 : len(op.Contents)-1]
	result := make([]PayoutCandidateSimulationResult, 0)

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
		// rebuild op for estimates
		op, err := buildOpForEstimation(ctx, []PayoutCandidateWithBondAmountAndFee{batch[i]}, false)
		if err != nil {
			return nil, err
		}

		feeBuffer := ctx.configuration.PayoutConfiguration.TxFeeBuffer
		if batch[i].Recipient.IsContract() {
			feeBuffer = ctx.configuration.PayoutConfiguration.KtTxFeeBuffer
		}
		common.InjectLimits(op, []tezos.Limits{{
			GasLimit:     p.GasUsed + ctx.configuration.PayoutConfiguration.TxGasLimitBuffer,
			StorageLimit: utils.CalculateStorageLimit(p),
			Fee:          p.Fee,
		}})

		txSerializationGas := (serializationGas * int64(len(bytes))) / int64(totalBytes)
		result = append(result, PayoutCandidateSimulationResult{
			AllocationBurn: p.AllocationBurn,
			StorageBurn:    p.StorageBurn,
			OpLimits: &common.OpLimits{
				GasLimit:              p.GasUsed + ctx.configuration.PayoutConfiguration.TxGasLimitBuffer,
				StorageLimit:          utils.CalculateStorageLimit(p),
				TransactionFee:        utils.EstimateTransactionFee(op, []int64{p.GasUsed + txSerializationGas + ctx.configuration.PayoutConfiguration.TxDeserializationGasBuffer}, feeBuffer),
				SerializationGasLimit: txSerializationGas + ctx.configuration.PayoutConfiguration.TxDeserializationGasBuffer,
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

	standardTxs := lo.Filter(candidates, func(p PayoutCandidateWithBondAmountAndFee, _ int) bool {
		return !p.Recipient.IsContract()
	})
	contractTxs := lo.Filter(candidates, func(p PayoutCandidateWithBondAmountAndFee, _ int) bool {
		return p.Recipient.IsContract()
	})
	faTxs := filterCandidatesByTxKind(contractTxs, []enums.EPayoutTransactionKind{enums.PAYOUT_TX_KIND_FA1_2, enums.PAYOUT_TX_KIND_FA1_2})
	others := rejectCandidatesByTxKind(contractTxs, []enums.EPayoutTransactionKind{enums.PAYOUT_TX_KIND_FA1_2, enums.PAYOUT_TX_KIND_FA1_2})

	batches := splitIntoBatches(others, TX_BATCH_CAPACITY)
	batches = append(batches, splitIntoBatches(faTxs, TX_BATCH_CAPACITY)...)
	batches = append(batches, splitIntoBatches(standardTxs, TX_BATCH_CAPACITY)...)

	batchesSimulated := lo.Map(batches, func(batch []PayoutCandidateWithBondAmountAndFee, index int) []PayoutCandidateSimulated {
		simulationResults, err := estimateBatchFees(batch, ctx)
		if err != nil {
			log.Tracef("failed to estimate tx costs of batch n.%d (falling back to one by one estimate) - %s", index, err.Error())
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
