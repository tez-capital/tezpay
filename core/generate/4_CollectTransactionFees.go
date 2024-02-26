package generate

import (
	"errors"
	"fmt"
	"os"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
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

func estimateBatchFees(batch []PayoutCandidateWithBondAmountAndFee, ctx *PayoutGenerationContext) ([]PayoutCandidateSimulationResult, error) {
	var (
		err     error
		receipt *rpc.Receipt
	)
	op, err := buildOpForEstimation(ctx, lo.Map(batch, func(c PayoutCandidateWithBondAmountAndFee, _ int) *PayoutCandidateWithBondAmountAndFee {
		return &c
	}), true)

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
		utils.PanicWithMetadata("partial estimate", "db8b7d5f4e34cc8b0fe42ecd43aa1ad7c8649bb3c0cd1f4889a4664a6c99910b", costs, batch)
	}

	serializationGas := costs[0].GasUsed - costs[len(costs)-1].GasUsed - ctx.StageData.BatchMetadataDeserializationGasLimit
	// remove first and last contents and limits it is only burn tx to measure serialization cost
	costs = costs[1 : len(costs)-1]
	if len(costs) != len(batch) {
		fmt.Println(len(costs), len(batch))
		os.Exit(1)
		utils.PanicWithMetadata("partial estimate", "d93813b9a34cf314a9dceb648736061ef499836c3a04b4be2239c0c7da2c3c47", costs, batch)
	}
	result := make([]PayoutCandidateSimulationResult, 0)
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
		op, err := buildOpForEstimation(ctx, []*PayoutCandidateWithBondAmountAndFee{&batch[i]}, false)
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

		totalTxGasUsed := p.GasUsed + txSerializationGas +
			ctx.configuration.PayoutConfiguration.TxGasLimitBuffer + // buffer for gas limit
			ctx.StageData.BatchMetadataDeserializationGasLimit + // potential gas used for deserialization if only one tx in batch
			ctx.configuration.PayoutConfiguration.TxDeserializationGasBuffer // buffer for deserialization gas limit

		result = append(result, PayoutCandidateSimulationResult{
			AllocationBurn: p.AllocationBurn,
			StorageBurn:    p.StorageBurn,
			OpLimits: &common.OpLimits{
				GasLimit:                p.GasUsed + ctx.configuration.PayoutConfiguration.TxGasLimitBuffer,
				StorageLimit:            utils.CalculateStorageLimit(p),
				TransactionFee:          utils.EstimateTransactionFee(op, []int64{totalTxGasUsed}, feeBuffer),
				DeserializationGasLimit: txSerializationGas + ctx.configuration.PayoutConfiguration.TxDeserializationGasBuffer,
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

	batches := splitIntoBatches(others, ctx.configuration.PayoutConfiguration.SimulationBatchSize)
	batches = append(batches, splitIntoBatches(faTxs, ctx.configuration.PayoutConfiguration.SimulationBatchSize)...)
	batches = append(batches, splitIntoBatches(standardTxs, ctx.configuration.PayoutConfiguration.SimulationBatchSize)...)

	batchesSimulated := lo.Map(batches, func(batch []PayoutCandidateWithBondAmountAndFee, index int) []PayoutCandidateSimulated {
		simulationResults, err := estimateBatchFees(batch, ctx)
		if err != nil {
			log.Tracef("failed to estimate tx costs of batch n.%d (falling back to one by one estimate) - %s", index, err.Error())
			return lo.Map(batch, func(candidate PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateSimulated {
				simulationResults, err := estimateBatchFees([]PayoutCandidateWithBondAmountAndFee{candidate}, ctx)
				if len(simulationResults) == 0 {
					err = errors.Join(fmt.Errorf("unexpected simulation results: %v", simulationResults), err)
				}
				if err != nil {
					log.Warnf("failed to estimate tx costs to '%s' (delegator: '%s', amount %d, kind '%s')\nerror: %s", candidate.Recipient, candidate.Source, candidate.BondsAmount.Int64(), candidate.TxKind, err.Error())
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
