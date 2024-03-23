package execute

import (
	"errors"
	"fmt"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func druRunExecutePayoutBatch(ctx *PayoutExecutionContext, batchId string, batch common.RecipeBatch) *common.BatchResult {
	log.Infof("dry running batch %s (%d transactions)", batchId, len(batch))
	opExecCtx, err := batch.ToOpExecutionContext(ctx.GetSigner(), ctx.GetTransactor())
	if err != nil {
		log.Warnf("batch %s - %s", batchId, err.Error())
		return common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), errors.Join(constants.ErrOperationContextCreationFailed, err))
	}
	return common.NewSuccessBatchResult(batch, tezos.ZeroOpHash)
}

func executePayoutBatch(ctx *PayoutExecutionContext, batchId string, batch common.RecipeBatch) *common.BatchResult {
	log.Infof("creating batch %s (%d transactions)", batchId, len(batch))
	opExecCtx, err := batch.ToOpExecutionContext(ctx.GetSigner(), ctx.GetTransactor())
	if err != nil {
		log.Warnf("batch %s - %s", batchId, err.Error())
		return common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), errors.Join(constants.ErrOperationContextCreationFailed, err))
	}

	log.Infof("broadcasting batch %s", batchId)
	err = opExecCtx.Dispatch(nil)
	if err != nil {
		log.Warnf("failed to broadcast batch %s - %s", batchId, utils.TryUnwrapRPCError(err).Error())
		return common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), errors.Join(constants.ErrOperationBroadcastFailed, err))
	}

	log.Infof("waiting for confirmation of batch %s (%s)", batchId, utils.GetOpReference(opExecCtx.GetOpHash(), ctx.configuration.Network.Explorer))
	ctx.protectedSection.Pause() // pause protected section to allow confirmation canceling
	err = opExecCtx.WaitForApply()
	ctx.protectedSection.Resume() // resume protected section
	if err != nil {
		log.Warnf("failed to apply batch %s - %s", batchId, err.Error())
		return common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), errors.Join(constants.ErrOperationConfirmationFailed, err))
	}

	log.Infof("batch %s - success", batchId)
	return common.NewSuccessBatchResult(batch, opExecCtx.GetOpHash())
}

func executePayouts(ctx *PayoutExecutionContext, options *common.ExecutePayoutsOptions) *PayoutExecutionContext {
	batchCount := len(ctx.StageData.Batches)
	batchesResults := make(common.BatchResults, 0)

	ctx.protectedSection.Start()
	log.Infof("paying out in %d batches", batchCount)
	reporter := ctx.GetReporter()
	for i, batch := range ctx.StageData.Batches {
		if err := reporter.ReportPayouts(batchesResults.ToReports()); err != nil {
			log.Warnf("failed to write partial report of payouts - %s", err.Error())
		}

		if ctx.protectedSection.Signaled() {
			batchesResults = append(batchesResults, *common.NewFailedBatchResult(batch, constants.ErrExecutePayoutsUserTerminated))
			ctx.AdminNotify("Payouts execution terminated by user")
			continue
		}

		batchId := fmt.Sprintf("%d/%d", i+1, batchCount)
		if options.DryRun {
			batchesResults = append(batchesResults, *druRunExecutePayoutBatch(ctx, batchId, batch))
		} else {
			batchesResults = append(batchesResults, *executePayoutBatch(ctx, batchId, batch))
		}
	}

	ctx.StageData.BatchResults = batchesResults
	failureDetected := false

	validPayoutReports := append(ctx.StageData.BatchResults.ToReports(), ctx.StageData.ReportsOfPastSuccesfulPayouts...)

	validAccumulatedPayouts := make([]common.PayoutReport, 0, len(ctx.AccumulatedPayouts))
	invalidAccumulatedPayouts := make([]common.PayoutRecipe, 0, len(ctx.AccumulatedPayouts))
	for _, payout := range ctx.AccumulatedPayouts {
		wasAccumulated, id, cycle := payout.GetAccumulatedPayoutDetails()
		if !wasAccumulated {
			continue
		}
		if !lo.ContainsBy(validPayoutReports, func(report common.PayoutReport) bool {
			return report.Cycle == cycle && report.Id == id
		}) {
			validAccumulatedPayouts = append(validAccumulatedPayouts, payout.ToPayoutReport())
		} else {
			invalidAccumulatedPayouts = append(invalidAccumulatedPayouts, payout)
		}
	}

	validPayoutReports = append(validPayoutReports, validAccumulatedPayouts...)
	if err := reporter.ReportPayouts(validPayoutReports); err != nil {
		log.Warnf("failed to report sent payouts - %s", err.Error())
		failureDetected = true
	}

	invalidPayoutReports := append(ctx.InvalidPayouts, invalidAccumulatedPayouts...)
	if err := reporter.ReportInvalidPayouts(invalidPayoutReports); err != nil {
		log.Warnf("failed to report invalid payouts - %s", err.Error())
		failureDetected = true
	}
	for _, blueprint := range ctx.PayoutBlueprints {
		if err := reporter.ReportCycleSummary(blueprint.Summary); err != nil {
			log.Warnf("failed to report cycle summary - %s", err.Error())
			failureDetected = true
		}
	}
	if !failureDetected {
		log.Info("all payouts reports written successfully")
	}

	ctx.protectedSection.Stop()

	paidDelegators := lo.Reduce(validPayoutReports, func(agg []tezos.Address, report common.PayoutReport, _ int) []tezos.Address {
		return append(agg, report.Delegator)
	}, []tezos.Address{})
	paidDelegators = lo.Uniq(paidDelegators)
	ctx.StageData.PaidDelegators = len(paidDelegators)
	return ctx
}

// NOTE: We should not return error here, because it could trigger a retry
func ExecutePayouts(ctx *PayoutExecutionContext, options *common.ExecutePayoutsOptions) (*PayoutExecutionContext, error) {
	return executePayouts(ctx, options), nil
}
