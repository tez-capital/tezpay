package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/core"
	reporter_engines "github.com/tez-capital/tezpay/engines/reporter"
	"github.com/tez-capital/tezpay/extension"
	"github.com/tez-capital/tezpay/state"
	"github.com/tez-capital/tezpay/utils"
)

var continualCmd = &cobra.Command{
	Use:   "continual",
	Short: "continual payout",
	Long:  "runs payout until stopped manually",
	Run: func(cmd *cobra.Command, args []string) {
		config, collector, signer, transactor := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		defer extension.CloseExtensions()
		initialCycle, _ := cmd.Flags().GetInt64(CYCLE_FLAG)
		endCycle, _ := cmd.Flags().GetInt64(END_CYCLE_FLAG)
		mixInContractCalls, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG)
		mixInFATransfers, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_FA_PAYOUTS_FLAG)
		forceConfirmationPrompt, _ := cmd.Flags().GetBool(FORCE_CONFIRMATION_PROMPT_FLAG)
		isDryRun, _ := cmd.Flags().GetBool(DRY_RUN_FLAG)

		fsReporter := reporter_engines.NewFileSystemReporter(config, &common.ReporterEngineOptions{
			DryRun: isDryRun,
		})

		if utils.IsTty() {
			assertRequireConfirmation("\n\n\t !!! ATTENTION !!!\n\nPreliminary testing has been conducted on the continual mode, but potential for undiscovered bugs still exists.\n Do you want to proceed?")
		}
		if forceConfirmationPrompt {
			if utils.IsTty() {
				slog.Info("you will be prompted for confirmation before each payout")
				time.Sleep(time.Second * 5)
			} else {
				slog.Error("force confirmation mode is not supported in non-interactive mode")
				os.Exit(EXIT_IVNALID_ARGS)
			}
		}

		if !state.Global.IsDonationPromptDisabled() && !config.IsDonatingToTezCapital() {
			assertRequireConfirmation("⚠️  With your current configuration you are not going to donate to tez.capital.😔 Do you want to proceed?")
		}

		monitor := assertRunWithResultAndErrorMessage(func() (common.CycleMonitor, error) {
			return collector.CreateCycleMonitor(common.CycleMonitorOptions{
				CheckFrequency:    10,
				NotificationDelay: rand.Int63n(config.PayoutConfiguration.MaximumDelayBlocks-config.PayoutConfiguration.MinimumDelayBlocks) + config.PayoutConfiguration.MinimumDelayBlocks,
			})
		}, EXIT_OPERTION_FAILED, "failed to init cycle monitor")

		// last completed cycle at the time we started continual mode on
		onchainCompletedCycle := assertRunWithResultAndErrorMessage(func() (int64, error) {
			return collector.GetLastCompletedCycle()
		}, EXIT_OPERTION_FAILED, "failed to get last completed cycle")
		// TODO: remove after AI
		startedAtCompletedCycle := onchainCompletedCycle
		// TODO: end remove after AI

		lastProcessedCycle := int64(onchainCompletedCycle)
		if initialCycle != 0 {
			if initialCycle > 0 {
				lastProcessedCycle = initialCycle - 1
			} else {
				lastProcessedCycle = onchainCompletedCycle + initialCycle
			}
		}
		var cycleToProcess int64

		completeCycle := func() {
			lastProcessedCycle = cycleToProcess
			slog.Info("cycle processed successfully", "cycle", cycleToProcess)
			slog.Info("=====================================================")
			if endCycle != 0 && lastProcessedCycle >= endCycle {
				slog.Info("end cycle reached, exiting")
				os.Exit(0)
			}
		}

		notifiedNewVersionAvailable := false

		startupProtocol := GetProtocolWithRetry(collector)
		if !config.Network.IgnoreProtocolChanges {
			slog.Info("Continual mode started in safe mode. In the event of a protocol change, TezPay will stop processing payouts and you will be notified.")
		}
		defer func() {
			notifyAdmin(config, fmt.Sprintf("Continual payouts stopped on cycle #%d", lastProcessedCycle+1))
		}()
		notifyAdmin(config, fmt.Sprintf("Continual payouts started on cycle #%d (tezpay %s, protocol %s)", lastProcessedCycle+1, constants.VERSION, startupProtocol))
		for {
			if lastProcessedCycle >= onchainCompletedCycle {
				slog.Info("waiting for next cycle to complete")
				var err error
				onchainCompletedCycle, err = monitor.WaitForNextCompletedCycle(lastProcessedCycle)
				if err != nil {
					if errors.Is(err, constants.ErrMonitoringCanceled) {
						slog.Info("cycle monitoring canceled")
						notifyAdmin(config, "Cycle monitoring canceled.")
					} else {
						slog.Error("failed to wait for next completed cycle", "error", err)
						notifyAdmin(config, "Failed to wait for next completed cycle.")
					}
					return
				}
			}
			if !config.Network.IgnoreProtocolChanges {
				slog.Debug("checking for protocol changes")
				currentProtocol := GetProtocolWithRetry(collector)
				if currentProtocol != startupProtocol {
					/// we can not exit here. Users may configure recover mechanism in case of crashes/exits so we really want to wait for the operator to take action
					slog.Warn("protocol changed, operator action required", "old_protocol", startupProtocol, "new_protocol", currentProtocol)
					notifyAdmin(config, fmt.Sprintf("Protocol changed from %s to %s, waiting for the operator to take action.", startupProtocol, currentProtocol))
					continue
				}
				// TODO: remove after AI
				if startedAtCompletedCycle < constants.FIRST_PARIS_AI_ACTIVATED_CYCLE /* if started before AI activation */ &&
					lastProcessedCycle+1 >= constants.FIRST_PARIS_AI_ACTIVATED_CYCLE /* and if we are going to process AI activated cycle */ {
					slog.Warn("AI activation cycle reached, stopping payouts until operator takes action")
					notifyAdmin(config, "AI activation cycle reached, stopping payouts until operator takes action")
					continue
				}
				// TODO: end remove after AI
			}

			defer extension.CloseScopedExtensions()
			cycleToProcess = lastProcessedCycle + 1

			if !notifiedNewVersionAvailable {
				if available, latest := checkForNewVersionAvailable(); available {
					notifyAdmin(config, fmt.Sprintf("New tezpay version available - %s", latest))
					notifiedNewVersionAvailable = true
				}
			}

			// refresh engine params - for protoocol upgrades
			if err := errors.Join(transactor.RefreshParams(), collector.RefreshParams()); err != nil {
				slog.Error("failed to check for protocol changes, retries in 5 minutes", "error", err)
				time.Sleep(time.Minute * 5)
				continue
			}

			slog.Info("=====================================================")
			slog.Info("processing cycle", "cycle", cycleToProcess)

			generationResult, err := core.GeneratePayouts(config, common.NewGeneratePayoutsEngines(collector, signer, notifyAdminFactory(config)),
				&common.GeneratePayoutsOptions{
					Cycle:                    cycleToProcess,
					WaitForSufficientBalance: true,
				})
			if err != nil {
				if errors.Is(err, constants.ErrNoCycleDataAvailable) {
					slog.Info("no data available for cycle, skipping", "cycle", cycleToProcess)
					completeCycle()
					continue
				}
				slog.Error("failed to generate payouts", "error", err)
				time.Sleep(time.Minute * 5)
				continue
			}

			slog.Info("checking reports of past payouts")
			preparationResult := assertRunWithResult(func() (*common.PreparePayoutsResult, error) {
				return core.PrepareCyclePayouts(generationResult, config, common.NewPreparePayoutsEngineContext(collector, signer, fsReporter, notifyAdminFactory(config)), &common.PreparePayoutsOptions{})
			}, EXIT_OPERTION_FAILED)

			if len(preparationResult.ValidPayouts) == 0 {
				slog.Info("nothing to pay out, skipping")
				completeCycle()
				continue
			}

			slog.Info("processing payouts", "valid", len(preparationResult.ValidPayouts), "invalid", len(preparationResult.InvalidPayouts), "accumulated", len(preparationResult.AccumulatedPayouts), "already_successfull", len(preparationResult.ReportsOfPastSuccesfulPayouts))

			if forceConfirmationPrompt && utils.IsTty() {
				cycles := []int64{generationResult.Cycle}
				utils.PrintPayouts(preparationResult.InvalidPayouts, fmt.Sprintf("Invalid - %s", utils.FormatCycleNumbers(cycles)), false)
				utils.PrintPayouts(preparationResult.AccumulatedPayouts, fmt.Sprintf("Accumulated - %s", utils.FormatCycleNumbers(cycles)), false)
				utils.PrintReports(preparationResult.ReportsOfPastSuccesfulPayouts, fmt.Sprintf("Already Successfull - %s", utils.FormatCycleNumbers(cycles)), true)
				utils.PrintPayouts(preparationResult.ValidPayouts, fmt.Sprintf("Valid - %s", utils.FormatCycleNumbers(cycles)), true)
				assertRequireConfirmation("Do you want to pay out above VALID payouts?")
			}

			slog.Info("executing payouts", "valid", len(preparationResult.ValidPayouts), "invalid", len(preparationResult.InvalidPayouts), "accumulated", len(preparationResult.AccumulatedPayouts), "already_successfull", len(preparationResult.ReportsOfPastSuccesfulPayouts))
			executionResult := assertRunWithResult(func() (*common.ExecutePayoutsResult, error) {
				return core.ExecutePayouts(preparationResult, config, common.NewExecutePayoutsEngineContext(signer, transactor, fsReporter, notifyAdminFactory(config)), &common.ExecutePayoutsOptions{
					MixInContractCalls: mixInContractCalls,
					MixInFATransfers:   mixInFATransfers,
					DryRun:             isDryRun,
				})
			}, EXIT_OPERTION_FAILED)

			// notify
			failedCount := lo.CountBy(executionResult.BatchResults, func(br common.BatchResult) bool { return !br.IsSuccess })
			if len(executionResult.BatchResults) > 0 && failedCount > 0 {
				slog.Error("failed operations detected", "failed", failedCount, "total", len(executionResult.BatchResults))
				time.Sleep(time.Minute * 5)
				continue
			}
			if silent, _ := cmd.Flags().GetBool(SILENT_FLAG); !silent {
				notifyPayoutsProcessedThroughAllNotificators(config, &generationResult.Summary)
			}
			completeCycle()
		}
	},
}

func init() {
	continualCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "initial cycle")
	continualCmd.Flags().Int64P(END_CYCLE_FLAG, "e", 0, "end cycle")
	continualCmd.Flags().Bool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG, false, "disables smart contract separation (mixes txs and smart contract calls within batches)")
	continualCmd.Flags().Bool(DISABLE_SEPERATE_FA_PAYOUTS_FLAG, false, "disables fa transfers separation (mixes txs and fa transfers within batches)")
	continualCmd.Flags().BoolP(FORCE_CONFIRMATION_PROMPT_FLAG, "a", false, "ask for confirmation on each payout")
	continualCmd.Flags().Bool(DRY_RUN_FLAG, false, "skips payout wallet balance check")

	RootCmd.AddCommand(continualCmd)
}
