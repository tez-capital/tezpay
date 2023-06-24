package generate

import (
	"errors"

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

func batchEstimate(payouts []PayoutCandidateWithBondAmountAndFee, ctx *PayoutGenerationContext) []PayoutCandidateSimulated {
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
		var (
			err     error
			receipt *rpc.Receipt
		)
		op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
		op.WithTTL(constants.MAX_OPERATION_TTL)
		for _, p := range batch {
			if err = common.InjectTransferContents(op, ctx.PayoutKey.Address(), &p); err != nil {
				break
			}
		}
		if err == nil {
			receipt, err = ctx.GetCollector().Simulate(op, ctx.PayoutKey)
		}
		if err != nil || !receipt.IsSuccess() {
			log.Tracef("failed to estimate tx costs of batch n.%d (falling back to one by one estimate)", index)
			return lo.Map(batch, func(candidate PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateSimulated {
				op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
				op.WithTTL(constants.MAX_OPERATION_TTL)
				err := common.InjectTransferContents(op, ctx.PayoutKey.Address(), &candidate)
				if err == nil {
					receipt, err = ctx.GetCollector().Simulate(op, ctx.PayoutKey)
				}
				if err != nil || !receipt.IsSuccess() {
					log.Warnf("failed to estimate tx costs to '%s' (delegator: '%s', amount %d, kind '%s')", candidate.Recipient, candidate.Source, candidate.BondsAmount.Int64(), candidate.TxKind)
					if receipt != nil && receipt.Error() != nil {
						errToLog := errors.Join(receipt.Error(), err).Error()
						if err != nil && receipt.Error().Error() == err.Error() {
							errToLog = err.Error()
						}
						log.Infof("estimating tx costs to '%s' failed because: %s", candidate.Recipient, errToLog)
					} else if err != nil {
						log.Infof("estimating tx costs to '%s' failed because: %s", candidate.Recipient, err.Error())
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
						// we use deserialization buffer as well because we use calculations here to substract fees from delegators
						// deserialization fee is later substracted from gas limit and added to first limit in the operation
						GasLimit:       costs.GasUsed + constants.TX_GAS_LIMIT_BUFFER + constants.TX_DESERIALIZATION_GAS_BUFFER,
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
					// we use deserialization buffer as well because we use calculations here to substract fees from delegators
					// deserialization fee is later substracted from gas limit and added to first limit in the operation
					GasLimit:       costs[index].GasUsed + constants.TX_GAS_LIMIT_BUFFER + constants.TX_DESERIALIZATION_GAS_BUFFER,
					StorageLimit:   utils.CalculateStorageLimit(costs[index]),
					TransactionFee: utils.EstimateContentFee(op.Contents[index], costs[index], op.Params, true),
				},
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
	simulatedPayouts := batchEstimate(candidates, ctx)

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
