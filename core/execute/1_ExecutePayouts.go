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
	logger.Info("dry running batch", "id", batchId, "tx_count", len(batch))
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
		return common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), errors.Join(constants.ErrOperationContextCreationFailed, err))
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
		if err := reporter.ReportPayouts(batchesResults.ToReports()); err != nil {
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
		logger.Warn("failed to report sent payouts", "error", err.Error())
		failureDetected = true
	}

	invalidPayoutReports := append(ctx.InvalidPayouts, invalidAccumulatedPayouts...)
	if err := reporter.ReportInvalidPayouts(invalidPayoutReports); err != nil {
		logger.Warn("failed to report invalid payouts", "error", err.Error())
		failureDetected = true
	}
	for _, blueprint := range ctx.PayoutBlueprints {
		if err := reporter.ReportCycleSummary(blueprint.Summary); err != nil {
			logger.Warn("failed to report cycle summary", "error", err.Error())
			failureDetected = true
		}
	}
	if !failureDetected {
		logger.Info("all payouts reports written successfully")
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
