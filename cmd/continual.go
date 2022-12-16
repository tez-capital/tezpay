package cmd

import (
	"fmt"
	"os"
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/ops"
	"github.com/alis-is/tezpay/core/payout"
	"github.com/alis-is/tezpay/core/reports"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var continualCmd = &cobra.Command{
	Use:   "continual",
	Short: "continual payout",
	Long:  "runs payout until stopped manually",
	Run: func(cmd *cobra.Command, args []string) {
		config, collector, signer, transactor := assertRunWithResult(loadConfigurationAndEngines, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		initialCycle, _ := cmd.Flags().GetInt64(CYCLE_FLAG)
		mixinContractCalls, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG)
		forceConfirmationPrompt, _ := cmd.Flags().GetBool(FORCE_CONFIRMATION_PROMPT_FLAG)

		assertRequireConfirmation("\n\n\t !!! WARNING !!!\\n\n Continual mode is not yet tested well enough and there are no payout confirmations.\n Do you want to proceed?")

		// TODO: remove
		disableConfirmationPrompt, _ := cmd.Flags().GetBool("disable-confirmation-prompt")
		forceConfirmationPrompt = true && !disableConfirmationPrompt

		monitor := assertRunWithResultAndErrFmt(func() (*common.CycleMonitor, error) {
			return collector.MonitorCycles(common.CycleMonitorOptions{
				CheckFrequency: 10,
			})
		}, EXIT_OPERTION_FAILED, "failed to init cycle monitor")

		currentCycle := <-monitor.Cycle
		lastProcessedCycle := int64(currentCycle - 1)
		if initialCycle != 0 {
			lastProcessedCycle = initialCycle - 1
		}
		var cycle int64

		completeCycle := func() {
			lastProcessedCycle = cycle
			log.Infof("================  CYCLE %d PROCESSED ===============", cycle)
		}

		for {
			if lastProcessedCycle >= currentCycle-1 {
				log.Info("looking for next cycle to pay out")
				currentCycle, ok := <-monitor.Cycle
				if !ok {
					os.Exit(0)
				}
				log.Infof("new cycle detected - %d", currentCycle)
				log.Debugf("current cycle %d, last processed %d", currentCycle, lastProcessedCycle)
				if lastProcessedCycle >= currentCycle {
					continue
				}
				cycle = currentCycle - 1
			} else {
				cycle = lastProcessedCycle + 1
			}

			log.Infof("====================  CYCLE %d  ====================", cycle)

			payoutBlueprint, err := payout.GeneratePayouts(signer.GetKey(), config, common.GeneratePayoutsOptions{
				Cycle:                    cycle,
				WaitForSufficientBalance: true,
				AdminNotify:              notifyAdminFactory(config),
				Engines: common.GeneratePayoutsEngines{
					Collector: collector,
				},
			})
			if err != nil {
				log.Errorf("failed to generate payout - %s, retries in 5 minutes", err.Error())
				time.Sleep(time.Minute * 5)
				continue
			}

			log.Info("checking past reports")
			reportResidues, err := loadPastPayoutReports(config.BakerPKH, payoutBlueprint.Cycle)
			if err != nil {
				log.Errorf("failed to read old payout reports from cycle #%d - %s, retries in 5 minutes", cycle, err.Error())
				time.Sleep(time.Minute * 5)
				continue
			}
			payouts, reportsOfPastSuccesfulPayouts := utils.FilterRecipesByReports(utils.OnlyValidPayouts(payoutBlueprint.Payouts), reportResidues, collector)

			log.Infof("processing %d valid payouts", len(payouts))

			if len(payouts) == 0 {
				log.Info("nothing to pay out, skipping")
				completeCycle()
				continue
			}

			if forceConfirmationPrompt {
				utils.PrintInvalidPayoutRecipes(payoutBlueprint.Payouts, payoutBlueprint.Cycle)
				utils.PrintReports(reportsOfPastSuccesfulPayouts, fmt.Sprintf("Already Successfull - #%d", payoutBlueprint.Cycle), true)
				utils.PrintValidPayoutRecipes(payouts, payoutBlueprint.Cycle)
				assertRequireConfirmation("Do you want to pay out above VALID payouts?")
			}

			limits, err := transactor.GetLimits()
			if err != nil {
				log.Errorf("ailed to get tezos chain limits - %s, retries in 5 minutes", err.Error())
				time.Sleep(time.Minute * 5)
				continue
			}

			var batches []ops.Batch
			if mixinContractCalls {
				batches = ops.SplitIntoBatches(payouts, limits)
			} else {
				contractBatches := ops.SplitIntoBatches(utils.FilterPayoutsByType(payouts, tezos.AddressTypeContract), limits)
				classicBatches := ops.SplitIntoBatches(utils.RejectPayoutsByType(payouts, tezos.AddressTypeContract), limits)
				batches = append(classicBatches, contractBatches...)
			}

			batchCount := len(batches)
			batchesResults := make(common.BatchResults, batchCount)

			log.Infof("paying out in %d batches", batchCount)
			for i, batch := range batches {
				// write past succesfuly
				warnIfFailedWithErrFmt(func() error { return reports.WritePayoutsReport(batchesResults.ToReports()) },
					"failed to write partial report of payouts - %s")

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

			finalPayoutReports := batchesResults.ToReports()
			finalPayoutReports = append(finalPayoutReports, reportsOfPastSuccesfulPayouts...)

			failureDetected := false
			failureDetected = warnIfFailedWithErrFmt(func() error { return reports.WriteInvalidPayoutRecipesReport(payoutBlueprint.Payouts) },
				"failed to write report of invalid payout recipes - %s")
			failureDetected = warnIfFailedWithErrFmt(func() error { return reports.WritePayoutsReport(finalPayoutReports) },
				"failed to write report of payouts - %s") || failureDetected
			failureDetected = warnIfFailedWithErrFmt(func() error { return reports.WriteCycleSummary(payoutBlueprint.Summary) },
				"failed to write cycle summary - %s") || failureDetected
			if !failureDetected {
				log.Info("all payouts reports written successfully")
			}

			// notify
			failedCount := lo.CountBy(batchesResults, func(br common.BatchResult) bool { return !br.IsSuccess })
			if len(batchesResults) > 0 && failedCount > 0 {
				log.Errorf("%d of operations failed, retries in 5 minutes", failedCount)
				time.Sleep(time.Minute * 5)
				continue
			}
			if silent, _ := cmd.Flags().GetBool(SILENT_FLAG); !silent {
				notifyPayoutsProcessedThroughAllNotificators(config, &payoutBlueprint.Summary)
			}
			completeCycle()
		}
	},
}

func init() {
	continualCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "initial cycle")
	continualCmd.Flags().Bool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG, false, "disables smart contract separation (mixes txs and smart contract calls within batches)")
	continualCmd.Flags().Bool(FORCE_CONFIRMATION_PROMPT_FLAG, false, "forces confirmation prompts for each payout")
	continualCmd.Flags().MarkHidden(FORCE_CONFIRMATION_PROMPT_FLAG)

	continualCmd.Flags().Bool("disable-confirmation-prompt", false, "disables confirmation prompts for each payout")
	continualCmd.Flags().MarkDeprecated("disable-confirmation-prompt", "this flag will be removed in the future")

	RootCmd.AddCommand(continualCmd)
}
