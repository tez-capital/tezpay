package stages

import (
	"blockwatch.cc/tzgo/codec"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/core/payout/common"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func collectTransactionFees(ctx common.Context) (result common.Context, err error) {
	configuration := ctx.GetConfiguration()
	candidates := ctx.StageData.PayoutCandidatesWithBondAmountAndFees

	log.Debug("simulating transactions to collect tx fees")
	// simulate
	simulatedPayouts := lo.Map(candidates, func(candidate common.PayoutCandidateWithBondAmountAndFee, _ int) common.PayoutCandidateSimulated {
		if candidate.Candidate.IsInvalid {
			return common.PayoutCandidateSimulated{
				Candidate: candidate.Candidate,
			}
		}
		// invalidate if zero
		if candidate.BondsAmount.IsZero() {
			candidate.Candidate.IsInvalid = true
			candidate.Candidate.InvalidBecause = enums.INVALID_PAYOUT_ZERO
			return common.PayoutCandidateSimulated{
				Candidate: candidate.Candidate,
			}
		}

		op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
		op.WithTransfer(candidate.Candidate.Recipient, candidate.BondsAmount.Int64())

		receipt, err := ctx.Collector.Simulate(op, ctx.PayoutKey)
		if err != nil {
			log.Warnf("failed to estimate tx costs to '%s' (delegator: '%s', amount %d)", candidate.Candidate.Recipient, candidate.Candidate.Source, candidate.BondsAmount.Int64())
			log.Debugf(err.Error())
			candidate.Candidate.IsInvalid = true
			candidate.Candidate.InvalidBecause = enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS
			return common.PayoutCandidateSimulated{
				Candidate: candidate.Candidate,
			}

		}

		costs := receipt.TotalCosts()

		return common.PayoutCandidateSimulated{
			Candidate:      candidate.Candidate,
			BondsAmount:    candidate.BondsAmount,
			Fee:            candidate.Fee,
			AllocationBurn: costs.AllocationBurn,
			StorageBurn:    costs.StorageBurn,
			OpLimits: &common.OpLimits{
				GasLimit:       costs.GasUsed + constants.GAS_LIMIT_BUFFER,
				StorageLimit:   utils.CalculateStorageLimit(costs),
				TransactionFee: utils.EstimateTransactionFee(op, receipt.Costs()),
			},
		}
	})

	simulatedPayouts = lo.Map(simulatedPayouts, func(candidate common.PayoutCandidateSimulated, _ int) common.PayoutCandidateSimulated {
		if candidate.Candidate.IsInvalid {
			return candidate
		}
		if configuration.PayoutConfiguration.IsPayingTxFee {
			candidate.BondsAmount = candidate.BondsAmount.Sub64(candidate.GetOperationFeesWithoutAllocation())
		}
		if configuration.PayoutConfiguration.IsPayingAllocationTxFee {
			candidate.BondsAmount = candidate.BondsAmount.Sub64(candidate.GetAllocationFee())
		}
		return candidate
	})
	ctx.StageData.PayoutCandidatesSimulated = simulatedPayouts
	return ctx, nil
}

var CollectTransactionFees = common.WrapStage(collectTransactionFees)
