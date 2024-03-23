package estimate

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
)

type EstimationContext struct {
	PayoutKey                            tezos.Key
	Collector                            common.CollectorEngine
	Configuration                        *configuration.RuntimeConfiguration
	BatchMetadataDeserializationGasLimit int64
}

func splitIntoBatches[T interface{}](candidates []T, capacity int) [][]T {
	batches := make([][]T, 0)
	if capacity == 0 {
		capacity = constants.DEFAULT_SIMULATION_TX_BATCH_SIZE
	}
	for offset := 0; offset < len(candidates); offset += capacity {
		batches = append(batches, lo.Slice(candidates, offset, offset+capacity))
	}

	return batches
}

func buildOpForEstimation[T common.TransferArgs](payoutKey tezos.Key, batch []T, injectBurnTransactions bool) (*codec.Op, error) {
	var err error
	op := codec.NewOp().WithSource(payoutKey.Address())
	op.WithTTL(constants.MAX_OPERATION_TTL)
	if injectBurnTransactions {
		op.WithTransfer(tezos.BurnAddress, 1)
	}
	for _, p := range batch {
		if err = common.InjectTransferContents(op, payoutKey.Address(), p); err != nil {
			break
		}
	}
	if injectBurnTransactions {
		op.WithTransfer(tezos.BurnAddress, 1)
	}
	return op, err
}

func estimateBatchFees[T common.TransferArgs](batch []T, ctx *EstimationContext) ([]*common.OpLimits, error) {
	var (
		err     error
		receipt *rpc.Receipt
	)
	op, err := buildOpForEstimation(ctx.PayoutKey, batch, true)

	if err != nil {
		return nil, err
	}
	receipt, err = ctx.Collector.Simulate(op, ctx.PayoutKey)
	if err != nil || (receipt != nil && !receipt.IsSuccess()) {
		if receipt != nil && receipt.Error() != nil && (err == nil || receipt.Error().Error() != err.Error()) {
			return nil, errors.Join(receipt.Error(), err)
		}
		return nil, err
	}

	costs := receipt.Op.Costs()
	if len(costs) < 3 {
		utils.PanicWithMetadata("partial estimate", "db8b7d5f4e34cc8b0fe42ecd43aa1ad7c8649bb3c0cd1f4889a4664a6c99910b", costs, batch)
	}

	serializationGas := costs[0].GasUsed - costs[len(costs)-1].GasUsed - ctx.BatchMetadataDeserializationGasLimit
	// remove first and last contents and limits it is only burn tx to measure serialization cost
	costs = costs[1 : len(costs)-1]
	if len(costs) != len(batch) {
		fmt.Println(len(costs), len(batch))
		os.Exit(1)
		utils.PanicWithMetadata("partial estimate", "d93813b9a34cf314a9dceb648736061ef499836c3a04b4be2239c0c7da2c3c47", costs, batch)
	}
	result := make([]*common.OpLimits, 0)
	// we intentionally caclculate total bytes on contents without first and last one (the ones used to determinie serialization fee)
	// to slightly increase per tx serialization fee as a buffer, it is likely going to be eaten be division anyway
	op.Contents = op.Contents[1 : len(op.Contents)-1]
	totalBytes := lo.Reduce(op.Contents, func(agg int, v codec.Operation, _ int) int {
		bytes, err := v.MarshalBinary()
		if err != nil {
			return agg
		}
		return agg + len(bytes)
	}, 0)

	for i, p := range costs {
		bytes, err := op.Contents[i].MarshalBinary()
		if err != nil {
			return nil, err
		}
		// rebuild op for estimates
		op, err := buildOpForEstimation(ctx.PayoutKey, []T{batch[i]}, false)
		if err != nil {
			return nil, err
		}

		feeBuffer := ctx.Configuration.PayoutConfiguration.TxFeeBuffer
		if batch[i].GetDestination().IsContract() || slices.Contains([]enums.EPayoutTransactionKind{enums.PAYOUT_TX_KIND_FA1_2, enums.PAYOUT_TX_KIND_FA2}, batch[i].GetTxKind()) {
			feeBuffer = ctx.Configuration.PayoutConfiguration.KtTxFeeBuffer
		}
		common.InjectLimits(op, []tezos.Limits{{
			GasLimit:     p.GasUsed + ctx.Configuration.PayoutConfiguration.TxGasLimitBuffer,
			StorageLimit: utils.CalculateStorageLimit(p),
			Fee:          p.Fee,
		}})

		txSerializationGas := (serializationGas * int64(len(bytes))) / int64(totalBytes)

		totalTxGasUsed := p.GasUsed + txSerializationGas +
			ctx.Configuration.PayoutConfiguration.TxGasLimitBuffer + // buffer for gas limit
			ctx.BatchMetadataDeserializationGasLimit + // potential gas used for deserialization if only one tx in batch
			ctx.Configuration.PayoutConfiguration.TxDeserializationGasBuffer // buffer for deserialization gas limit

		result = append(result, &common.OpLimits{
			GasLimit:                p.GasUsed + ctx.Configuration.PayoutConfiguration.TxGasLimitBuffer,
			StorageLimit:            utils.CalculateStorageLimit(p),
			TransactionFee:          utils.EstimateTransactionFee(op, []int64{totalTxGasUsed}, feeBuffer),
			DeserializationGasLimit: txSerializationGas + ctx.Configuration.PayoutConfiguration.TxDeserializationGasBuffer,
			AllocationBurn:          p.AllocationBurn,
			StorageBurn:             p.StorageBurn,
		})
	}

	return result, err
}

type EstimateResult[T common.TransferArgs] struct {
	Transaction T
	Result      *common.OpLimits
	Error       error
}

func EstimateTransactionFees[T common.TransferArgs](transactions []T, ctx *EstimationContext) []EstimateResult[T] {
	standardTxs := make([]T, 0, len(transactions))
	faTxs := make([]T, 0, len(transactions))
	otherTxs := make([]T, 0, len(transactions))

	for _, tx := range transactions {
		switch {
		case tx.GetTxKind() == enums.PAYOUT_TX_KIND_TEZ && !tx.GetDestination().IsContract():
			standardTxs = append(standardTxs, tx)
		case slices.Contains([]enums.EPayoutTransactionKind{enums.PAYOUT_TX_KIND_FA1_2, enums.PAYOUT_TX_KIND_FA2}, tx.GetTxKind()):
			faTxs = append(faTxs, tx)
		default:
			otherTxs = append(otherTxs, tx)
		}
	}

	batches := splitIntoBatches(otherTxs, ctx.Configuration.PayoutConfiguration.SimulationBatchSize)
	batches = append(batches, splitIntoBatches(faTxs, ctx.Configuration.PayoutConfiguration.SimulationBatchSize)...)
	batches = append(batches, splitIntoBatches(standardTxs, ctx.Configuration.PayoutConfiguration.SimulationBatchSize)...)

	simulationResults := lo.Map(batches, func(batch []T, index int) []EstimateResult[T] {
		simulationResults, err := estimateBatchFees(batch, ctx)
		if err != nil {
			return lo.Map(batch, func(candidate T, _ int) EstimateResult[T] {
				simulationResults, err := estimateBatchFees([]T{candidate}, ctx)
				if len(simulationResults) == 0 {
					err = errors.Join(fmt.Errorf("unexpected simulation results: %v", simulationResults), err)
				}
				if err != nil {
					return EstimateResult[T]{
						Transaction: candidate,
						Error:       err,
					}
				}

				return EstimateResult[T]{
					Transaction: candidate,
					Result:      simulationResults[0],
				}
			})
		}
		return lo.Map(batch, func(candidate T, index int) EstimateResult[T] {
			if index >= len(simulationResults) {
				panic("Partial estimate. This should never happen!")
			}
			return EstimateResult[T]{
				Transaction: candidate,
				Result:      simulationResults[index],
			}
		})
	})
	return lo.Flatten(simulationResults)
}
