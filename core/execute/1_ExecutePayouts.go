package execute

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/state"
	"github.com/tez-capital/tezpay/utils"
)

func buildBatchExecutionContext(ctx *PayoutExecutionContext, logger *slog.Logger, batchId string, batch common.RecipeBatch) (*common.OpExecutionContext, error) {
	logger = logger.With("batch_id", batchId)
	if state.Global.GetWantsOutputJson() {
		logger.Info("creating batch", "recipes", batch, "phase", "executing_batch")
	} else {
		logger.Info("creating batch", "tx_count", len(batch), "phase", "executing_batch")
	}
	return batch.ToOpExecutionContext(ctx.GetSigner(), ctx.GetTransactor())
}

func druRunExecuteBatch(ctx *PayoutExecutionContext, logger *slog.Logger, batchId string, opExecCtx *common.OpExecutionContext) *common.BatchResult {
	logger.Info("broadcasting batch")
	time.Sleep(2 * time.Second)
	logger.Info("waiting for confirmation", "op_reference", utils.GetOpReference(opExecCtx.GetOpHash(), ctx.GetConfiguration().Network.Explorer), "op_hash", opExecCtx.GetOpHash(), "phase", "batch_waiting_for_confirmation")
	time.Sleep(4 * time.Second)
	logger.Info("batch successful", "phase", "batch_execution_finished")
	return opExecCtx.AsSuccessBatchResult()
}

func executeBatch(ctx *PayoutExecutionContext, logger *slog.Logger, batchId string, opExecCtx *common.OpExecutionContext) *common.BatchResult {
	logger.Info("broadcasting batch")
	err := opExecCtx.Dispatch(nil)
	if err != nil {
		logger.Warn("failed to broadcast batch", "error", err.Error(), "phase", "batch_execution_finished")
		return opExecCtx.AsFailedBatchResult(errors.Join(constants.ErrOperationBroadcastFailed, err))
	}

	logger.Info("waiting for confirmation", "op_reference", utils.GetOpReference(opExecCtx.GetOpHash(), ctx.GetConfiguration().Network.Explorer), "op_hash", opExecCtx.GetOpHash(), "phase", "batch_waiting_for_confirmation")
	ctx.protectedSection.Pause() // pause protected section to allow confirmation canceling
	err = opExecCtx.WaitForApply()
	ctx.protectedSection.Resume() // resume protected section
	if err != nil {
		logger.Warn("failed to apply batch", "error", err.Error(), "phase", "batch_execution_finished")
		return opExecCtx.AsFailedBatchResult(errors.Join(constants.ErrOperationConfirmationFailed, err))
	}

	logger.Info("batch successful", "phase", "batch_execution_finished")
	return opExecCtx.AsSuccessBatchResult()
}

func executePayouts(ctx *PayoutExecutionContext, options *common.ExecutePayoutsOptions) *PayoutExecutionContext {
	logger := ctx.logger
	batchCount := len(ctx.StageData.Batches)
	batchesResults := make(common.BatchResults, 0)

	ctx.protectedSection.Start()
	logger.Info("paying out", "batches_count", batchCount, "phase", "batch_execution_start")
	reporter := ctx.GetReporter()
	for i, batch := range ctx.StageData.Batches {
		if err := reporter.ReportPayouts(append(batchesResults.ToIndividualReports(), ctx.StageData.ReportsOfPastSuccesfulPayouts...)); err != nil {
			logger.Warn("failed to write partial report of payouts", "error", err.Error())
		}

		if ctx.protectedSection.Signaled() {
			batchesResults = append(batchesResults, common.NewFailedBatchResult(batch, constants.ErrExecutePayoutsUserTerminated))
			ctx.AdminNotify("Payouts execution terminated by user")
			continue
		}

		batchId := fmt.Sprintf("%d/%d", i+1, batchCount)
		batchExecutionContext, err := buildBatchExecutionContext(ctx, logger, batchId, batch)
		if err != nil {
			logger.Warn("failed to create operation execution context", "error", err.Error(), "phase", "batch_execution_finished")
			batchesResults = append(batchesResults, common.NewFailedBatchResult(batch, errors.Join(constants.ErrOperationContextCreationFailed, err)))
			continue
		}

		tmpReport := append(batchesResults.ToIndividualReports(), ctx.StageData.ReportsOfPastSuccesfulPayouts...)
		// we record payouts as "in progress" before execution to avoid double spending in case of crash
		tmpReport = append(tmpReport, batchExecutionContext.AsFailedBatchResult(constants.ErrPayoutRecordedBeforeExecution).ToIndividualReports()...)
		if err := reporter.ReportPayouts(tmpReport); err != nil {
			batchesResults = append(batchesResults, batchExecutionContext.AsFailedBatchResult(errors.Join(constants.ErrFailedToRecordPayoutsBeforeExecution, err)))
			continue
		}

		if options.DryRun {
			batchesResults = append(batchesResults, druRunExecuteBatch(ctx, logger, batchId, batchExecutionContext))
		} else {
			batchesResults = append(batchesResults, executeBatch(ctx, logger, batchId, batchExecutionContext))
		}
	}

	failureDetected := false
	successfulPayoutReports := append(batchesResults.ToIndividualReports(), ctx.StageData.ReportsOfPastSuccesfulPayouts...)
	if err := reporter.ReportPayouts(successfulPayoutReports); err != nil {
		logger.Warn("!!! failed to report sent payouts !!!", "error", err.Error())
		failureDetected = true
	}
	ctx.StageData.BatchResults = batchesResults

	invalidReports := lo.Map(ctx.InvalidPayouts, func(pr common.PayoutRecipe, _ int) common.PayoutReport {
		return pr.ToPayoutReport()
	})

	if err := reporter.ReportInvalidPayouts(invalidReports); err != nil {
		logger.Warn("failed to report invalid payouts", "error", err.Error())
		failureDetected = true
	}

	summary := utils.GeneratePayoutSummary(ctx.PayoutBlueprints, append(successfulPayoutReports, invalidReports...))
	for cycle, cycleSummary := range summary.CycleSummaries {
		if err := reporter.ReportCycleSummary(cycle, cycleSummary); err != nil {
			logger.Warn("failed to report cycle summary", "error", err.Error())
			failureDetected = true
		}
	}
	if !failureDetected {
		logger.Info("all payouts reports written successfully")
	}

	ctx.StageData.Summary = *summary
	ctx.protectedSection.Stop()

	return ctx
}

// NOTE: We should not return error here, because it could trigger a retry
func ExecutePayouts(ctx *PayoutExecutionContext, options *common.ExecutePayoutsOptions) (*PayoutExecutionContext, error) {
	return executePayouts(ctx, options), nil
}
