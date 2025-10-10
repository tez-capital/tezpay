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
	"github.com/trilitech/tzgo/tezos"
)

func druRunExecutePayoutBatch(ctx *PayoutExecutionContext, logger *slog.Logger, batchId string, batch common.RecipeBatch) *common.BatchResult {
	logger = logger.With("batch_id", batchId)
	if state.Global.GetWantsOutputJson() {
		logger.Info("creating batch", "recipes", batch, "phase", "executing_batch")
	} else {
		logger.Info("creating batch", "tx_count", len(batch), "phase", "executing_batch")
	}
	opExecCtx, err := batch.ToOpExecutionContext(ctx.GetSigner(), ctx.GetTransactor())
	if err != nil {
		logger.Warn("failed to create operation execution context", "id", batchId, "error", err.Error(), "phase", "batch_execution_finished")
		return common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), errors.Join(constants.ErrOperationContextCreationFailed, err))
	}
	logger.Info("broadcasting batch")
	time.Sleep(2 * time.Second)
	logger.Info("waiting for confirmation", "op_reference", utils.GetOpReference(opExecCtx.GetOpHash(), ctx.GetConfiguration().Network.Explorer), "op_hash", opExecCtx.GetOpHash(), "phase", "batch_waiting_for_confirmation")
	time.Sleep(4 * time.Second)
	logger.Info("batch successful", "phase", "batch_execution_finished")
	return common.NewSuccessBatchResult(batch, tezos.ZeroOpHash)
}

func executePayoutBatch(ctx *PayoutExecutionContext, logger *slog.Logger, batchId string, batch common.RecipeBatch) *common.BatchResult {
	logger = logger.With("batch_id", batchId)
	if state.Global.GetWantsOutputJson() {
		logger.Info("creating batch", "recipes", batch, "phase", "executing_batch")
	} else {
		logger.Info("creating batch", "tx_count", len(batch), "phase", "executing_batch")
	}
	opExecCtx, err := batch.ToOpExecutionContext(ctx.GetSigner(), ctx.GetTransactor())
	if err != nil {
		logger.Warn("failed to create operation execution context", "error", err.Error(), "phase", "batch_execution_finished")
		opHash := tezos.ZeroOpHash
		if opExecCtx != nil {
			opHash = opExecCtx.GetOpHash()
		}
		return common.NewFailedBatchResultWithOpHash(batch, opHash, errors.Join(constants.ErrOperationContextCreationFailed, err))
	}

	logger.Info("broadcasting batch")
	err = opExecCtx.Dispatch(nil)
	if err != nil {
		logger.Warn("failed to broadcast batch", "error", err.Error(), "phase", "batch_execution_finished")
		return common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), errors.Join(constants.ErrOperationBroadcastFailed, err))
	}

	logger.Info("waiting for confirmation", "op_reference", utils.GetOpReference(opExecCtx.GetOpHash(), ctx.GetConfiguration().Network.Explorer), "op_hash", opExecCtx.GetOpHash(), "phase", "batch_waiting_for_confirmation")
	ctx.protectedSection.Pause() // pause protected section to allow confirmation canceling
	err = opExecCtx.WaitForApply()
	ctx.protectedSection.Resume() // resume protected section
	if err != nil {
		logger.Warn("failed to apply batch", "error", err.Error(), "phase", "batch_execution_finished")
		return common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), errors.Join(constants.ErrOperationConfirmationFailed, err))
	}

	logger.Info("batch successful", "phase", "batch_execution_finished")
	return common.NewSuccessBatchResult(batch, opExecCtx.GetOpHash())
}

func executePayouts(ctx *PayoutExecutionContext, options *common.ExecutePayoutsOptions) *PayoutExecutionContext {
	logger := ctx.logger
	batchCount := len(ctx.StageData.Batches)
	batchesResults := make(common.BatchResults, 0)

	ctx.protectedSection.Start()
	logger.Info("paying out", "batches_count", batchCount, "phase", "batch_execution_start")
	reporter := ctx.GetReporter()
	for i, batch := range ctx.StageData.Batches {
		if err := reporter.ReportPayouts(append(batchesResults.ToReports(), ctx.StageData.ReportsOfPastSuccesfulPayouts...)); err != nil {
			logger.Warn("failed to write partial report of payouts", "error", err.Error())
		}

		if ctx.protectedSection.Signaled() {
			batchesResults = append(batchesResults, *common.NewFailedBatchResult(batch, constants.ErrExecutePayoutsUserTerminated))
			ctx.AdminNotify("Payouts execution terminated by user")
			continue
		}

		batchId := fmt.Sprintf("%d/%d", i+1, batchCount)
		if options.DryRun {
			batchesResults = append(batchesResults, *druRunExecutePayoutBatch(ctx, logger, batchId, batch))
		} else {
			batchesResults = append(batchesResults, *executePayoutBatch(ctx, logger, batchId, batch))
		}
	}

	failureDetected := false
	successfulPayoutReports := append(ctx.StageData.BatchResults.ToReports(), ctx.StageData.ReportsOfPastSuccesfulPayouts...)
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
