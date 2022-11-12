package cmd

import (
	"fmt"
	"os"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/ops"
	"github.com/alis-is/tezpay/core/payout"
	"github.com/alis-is/tezpay/core/reports"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var payCmd = &cobra.Command{
	Use:   "pay",
	Short: "manual payout",
	Long:  "runs manual payout",
	Run: func(cmd *cobra.Command, args []string) {
		config, _, signer, transactor := assertRunWithResult(loadConfigurationAndEngines, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		cycle, _ := cmd.Flags().GetInt64(CYCLE_FLAG)
		confirmed, _ := cmd.Flags().GetBool(CONFIRM_FLAG)
		mixinContractCalls, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG)

		var payoutBlueprint *common.CyclePayoutBlueprint
		fromFile, _ := cmd.Flags().GetString(TO_FILE_FLAG)
		if fromFile != "" {
			payoutBlueprint = assertRunWithResult(func() (*common.CyclePayoutBlueprint, error) {
				return loadPayoutBlueprintFromFile(fromFile)
			}, EXIT_PAYOUTS_READ_FAILURE)
		} else {
			payoutBlueprint = assertRunWithResult(func() (*common.CyclePayoutBlueprint, error) {
				return payout.GeneratePayoutsWithPayoutAddress(signer.GetKey(), cycle, config)
			}, EXIT_OPERTION_FAILED)
		}
		log.Info("checking past reports")
		reportResidues := assertRunWithResultAndErrFmt(func() ([]common.PayoutReport, error) {
			return loadPastPayoutReports(config.BakerPKH, payoutBlueprint.Cycle)
		}, EXIT_PAYOUT_REPORTS_PARSING_FAULURE, "Failed to read old payout reports from cycle #%d - %s")
		payouts, reportsOfPastSuccesfulPayouts := utils.FilterRecipesByReports(utils.OnlyValidPayouts(payoutBlueprint.Payouts), reportResidues, nil)

		if state.Global.GetWantsOutputJson() {
			utils.PrintPayoutsAsJson(reportsOfPastSuccesfulPayouts)
			utils.PrintPayoutsAsJson(payouts)
		} else {
			utils.PrintInvalidPayoutRecipes(payoutBlueprint.Payouts, payoutBlueprint.Cycle)
			utils.PrintReports(reportsOfPastSuccesfulPayouts, fmt.Sprintf("Already Successfull - #%d", payoutBlueprint.Cycle), true)
			utils.PrintValidPayoutRecipes(payouts, payoutBlueprint.Cycle)
		}

		if len(payouts) == 0 {
			log.Info("nothing to pay out")
			notificator, _ := cmd.Flags().GetString(NOTIFICATOR_FLAG)
			if notificator != "" { // rerun notification through notificator if specified manually
				notifyPayoutsProcessed(config, &payoutBlueprint.Summary, notificator)
			}
			os.Exit(0)
		}

		if !confirmed {
			assertRequireConfirmation("Do you want to pay out above VALID payouts?")
		}

		log.Info("executing payout")
		limits := assertRunWithResultAndErrFmt(transactor.GetLimits, EXIT_OPERTION_FAILED, "failed to get tezos chain limits - %s")

		var batches []ops.Batch
		if mixinContractCalls {
			batches = ops.SplitIntoBatches(payouts, limits)
		} else {
			contractBatches := ops.SplitIntoBatches(utils.FilterPayoutsByType(payouts, tezos.AddressTypeContract), limits)
			classicBatches := ops.SplitIntoBatches(utils.RejectPayoutsByType(payouts, tezos.AddressTypeContract), limits)
			batches = append(classicBatches, contractBatches...)
		}

		batchCount := len(batches)
		batchesResults := make([]common.BatchResult, batchCount)

		log.Infof("paying out in %d batches", batchCount)
		for i, batch := range batches {
			log.Infof("creating batch n.%d of %d (%d transactions)", i+1, batchCount, len(batch))
			opExecCtx, err := batch.ToOpExecutionContext(signer, transactor)
			if err != nil {
				log.Warnf("batch n.%d - %s", i+1, err.Error())
				batchesResults[i] = *common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), fmt.Errorf("failed to create operation context - %s", err.Error()))
				continue
			}
			log.Infof("broadcasting batch n.%d", i+1)
			err = opExecCtx.Dispatch(nil)
			if err != nil {
				log.Warnf("batch n.%d - %s", i+1, err.Error())
				batchesResults[i] = *common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), fmt.Errorf("failed to broadcast - %s", err.Error()))
				continue
			}

			log.Infof("waiting for confirmation of batch n.%d (%s)", i+1, utils.GetOpReference(opExecCtx.GetOpHash(), config.Network.Explorer))
			err = opExecCtx.WaitForApply()
			if err != nil {
				log.Warnf("batch n.%d - %s", i+1, err.Error())
				batchesResults[i] = *common.NewFailedBatchResultWithOpHash(batch, opExecCtx.GetOpHash(), fmt.Errorf("failed to confirm - %s", err.Error()))
				continue
			}

			log.Infof("batch n.%d - success", i+1)
			batchesResults[i] = *common.NewSuccessBatchResult(batch, opExecCtx.GetOpHash())
		}

		finalPayoutReports := lo.Flatten(lo.Map(batchesResults, func(br common.BatchResult, _ int) []common.PayoutReport { return br.ToReports() }))
		finalPayoutReports = append(finalPayoutReports, reportsOfPastSuccesfulPayouts...)
		// write reports
		reportToStdout, _ := cmd.Flags().GetBool(REPORT_TO_STDOUT)
		if !reportToStdout {
			failureDetected := false
			failureDetected = warnIfFailedWithErrFmt(func() error { return reports.WriteInvalidPayoutRecipesReport(payoutBlueprint.Payouts) },
				"failed to write report of invalid payouts - %s")
			failureDetected = warnIfFailedWithErrFmt(func() error { return reports.WritePayoutsReport(finalPayoutReports) },
				"failed to write report of invalid payouts - %s") || failureDetected
			failureDetected = warnIfFailedWithErrFmt(func() error { return reports.WriteCycleSummary(payoutBlueprint.Summary) },
				"failed to write cycle summary - %s") || failureDetected
			if !failureDetected {
				log.Info("all payouts reports written successfully")
			}
		} else {
			assertRunWithParam(printPayoutCycleReport, &common.PayoutCycleReport{
				Cycle:   payoutBlueprint.Cycle,
				Invalid: utils.OnlyInvalidPayouts(payoutBlueprint.Payouts),
				Payouts: finalPayoutReports,
				Sumary:  &payoutBlueprint.Summary,
			}, EXIT_CYCLE_PAYOUT_REPORT_MARSHAL_FAILURE)
		}

		// notify
		failedCount := lo.CountBy(batchesResults, func(br common.BatchResult) bool { return !br.IsSuccess })
		if len(batchesResults) > 0 && failedCount > 0 {
			log.Errorf("%d of operations failed", failedCount)
			os.Exit(EXIT_OPERTION_FAILED)
		}
		if silent, _ := cmd.Flags().GetBool(SILENT_FLAG); !silent {
			notifyPayoutsProcessedThroughAllNotificators(config, &payoutBlueprint.Summary)
		}
		utils.PrintBatchResults(batchesResults, fmt.Sprintf("Results of #%d", payoutBlueprint.Cycle), config.Network.Explorer)
	},
}

func init() {
	payCmd.Flags().Bool(CONFIRM_FLAG, false, "automatically confirms generated payouts")
	payCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "cycle to generate payouts for")
	payCmd.Flags().Bool(REPORT_TO_STDOUT, false, "prints them to stdout (wont write to file)")
	payCmd.Flags().String(FROM_FILE_FLAG, "", "loads payouts from file instead of generating on the fly")
	payCmd.Flags().Bool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG, false, "disables smart contract separation (mixes txs and smart contract calls within batches)")
	payCmd.Flags().BoolP(SILENT_FLAG, "s", false, "suppresses notifications")
	payCmd.Flags().String(NOTIFICATOR_FLAG, "", "Notify through specific notificator")

	RootCmd.AddCommand(payCmd)
}
