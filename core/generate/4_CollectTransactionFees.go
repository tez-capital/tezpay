package generate

import (
	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/core/estimate"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/tezos"
)

func CollectTransactionFees(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (result *PayoutGenerationContext, err error) {
	candidates := ctx.StageData.PayoutCandidatesWithBondAmountAndFees
	logger := ctx.logger.With("phase", "collect_transaction_fees")
	logger.Info("collecting transaction fees")
	// presim validation
	candidates = lo.Map(candidates, func(candidate PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateWithBondAmountAndFee {
		validationContext := candidate.ToValidationContext(ctx)
		return *validationContext.Validate(
			TxKindValidator,
		).ToPresimPayoutCandidate()
	})

	logger.Debug("simulating transactions to collect tx fees")
	validCandidates := make([]PayoutCandidateWithBondAmountAndFee, 0)
	invalidCandidates := make([]PayoutCandidateSimulated, 0)
	for _, candidate := range candidates {
		if candidate.IsInvalid {
			invalidCandidates = append(invalidCandidates, PayoutCandidateSimulated{
				PayoutCandidateWithBondAmountAndFee: candidate,
			})
		} else {
			validCandidates = append(validCandidates, candidate)
		}
	}

	simulatioResults := estimate.EstimateTransactionFees(utils.MapToPointers(validCandidates), &estimate.EstimationContext{
		PayoutKey:                            ctx.PayoutKey,
		Collector:                            ctx.GetCollector(),
		Configuration:                        ctx.configuration,
		BatchMetadataDeserializationGasLimit: ctx.StageData.BatchMetadataDeserializationGasLimit,
	})

	simulatedPayouts := lo.Map(simulatioResults, func(result estimate.EstimateResult[*PayoutCandidateWithBondAmountAndFee], _ int) PayoutCandidateSimulated {
		if result.Transaction.IsInvalid { // we don't collect fees from non-tez payouts
			return PayoutCandidateSimulated{
				PayoutCandidateWithBondAmountAndFee: *result.Transaction,
			}
		}
		if result.Error != nil {
			logger.Warn("failed to estimate tx costs", "recipient", result.Transaction.Recipient, "delegator", result.Transaction.Source, "amount", result.Transaction.BondsAmount.Int64(), "kind", result.Transaction.TxKind, "error", result.Error)
			result.Transaction.IsInvalid = true
			result.Transaction.InvalidBecause = enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS
			return PayoutCandidateSimulated{
				PayoutCandidateWithBondAmountAndFee: *result.Transaction,
			}
		}

		candidate := PayoutCandidateSimulated{
			PayoutCandidateWithBondAmountAndFee: *result.Transaction,
			SimulationResult:                    result.Result,
		}
		if candidate.TxKind == enums.PAYOUT_TX_KIND_TEZ {
			bondsAmountBeforeFees := candidate.BondsAmount
			utils.AssertZAmountPositiveOrZero(bondsAmountBeforeFees)

			txFee := candidate.SimulationResult.GetOperationFeesWithoutAllocation()
			allocationFee := candidate.SimulationResult.GetAllocationFee()

			if !candidate.IsBakerPayingTxFee {
				candidate.BondsAmount = candidate.BondsAmount.Sub64(txFee)
				candidate.TxFeeCollected = true
			}
			if !candidate.IsBakerPayingAllocationTxFee {
				candidate.BondsAmount = candidate.BondsAmount.Sub64(allocationFee)
				candidate.AllocationFeeCollected = true
			}

			if candidate.BondsAmount.IsNeg() || candidate.BondsAmount.IsZero() {
				candidate.IsInvalid = true
				candidate.BondsAmount = tezos.Zero
				candidate.InvalidBecause = enums.INVALID_NOT_ENOUGH_BONDS_FOR_TX_FEES
				candidate.Fee = candidate.Fee.Add(bondsAmountBeforeFees)                                 // collect the whole bonds amount as fee if not enough for tx fees
				ctx.StageData.BakerFeesAmount = ctx.StageData.BakerFeesAmount.Add(bondsAmountBeforeFees) // collect fees if invalid
			}
			utils.AssertZAmountPositiveOrZero(candidate.BondsAmount)
		}

		return candidate
	})

	ctx.StageData.PayoutCandidatesSimulated = append(invalidCandidates, simulatedPayouts...)
	return ctx, nil
}
