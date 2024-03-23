package generate

import (
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/core/estimate"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

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
			log.Warnf("failed to estimate tx costs to '%s' (delegator: '%s', amount %d, kind '%s')\nerror: %s", result.Transaction.Recipient, result.Transaction.Source, result.Transaction.BondsAmount.Int64(), result.Transaction.TxKind, result.Error.Error())
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
			if !candidate.IsBakerPayingTxFee {
				candidate.BondsAmount = candidate.BondsAmount.Sub64(candidate.SimulationResult.GetOperationFeesWithoutAllocation())
				candidate.TxFeeCollected = true
			}
			if !candidate.IsBakerPayingAllocationTxFee {
				candidate.BondsAmount = candidate.BondsAmount.Sub64(candidate.SimulationResult.GetAllocationFee())
				candidate.AllocationFeeCollected = true
			}
		}

		return candidate
	})

	ctx.StageData.PayoutCandidatesSimulated = append(invalidCandidates, simulatedPayouts...)
	return ctx, nil
}
