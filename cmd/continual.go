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

var (
	onchainCompletedCycle int64
	lastProcessedCycle    int64
	cycleToProcess        int64
	endCycle              int64
)

func processCycleInContinualMode(context *configurationAndEngines, forceConfirmationPrompt bool, mixInContractCalls bool, mixInFATransfers bool, isDryRun bool, silent bool, payoutPeriod int64) (processed bool) {
	processed = true
	retry := func() bool {
		processed = false
		return false
	}

	cycleToProcess = lastProcessedCycle + 1
	cycles, isEndOfThePeriod := getCyclesInCompletedPeriod(cycleToProcess, payoutPeriod)
	if !isEndOfThePeriod {
		slog.Info("cycle is not at the end of the specified payout period, skipping", "cycle", cycleToProcess, "payout_period", payoutPeriod)
		lastProcessedCycle = cycleToProcess
		return
	}

	defer func() { // complete cycle
		switch {
		case processed:
			lastProcessedCycle = cycleToProcess
			slog.Info("cycle processed successfully", "cycle", cycleToProcess)
			slog.Info("===================== PROCESSING -END- =====================")
			extension.CloseScopedExtensions()
			if endCycle != 0 && lastProcessedCycle >= endCycle {
				slog.Info("end cycle reached, exiting")
				os.Exit(0)
			}
		default:
			slog.Info("cycle processing failed, retrying in 5 minutes")
			time.Sleep(time.Minute * 5) // wait for a while before retry
		}
	}()

	config, collector, signer, transactor := context.Unwrap()
	fsReporter := reporter_engines.NewFileSystemReporter(config, &common.ReporterEngineOptions{
		DryRun: isDryRun,
	})

	// refresh engine params - for protocol upgrades
	if err := errors.Join(transactor.RefreshParams(), collector.RefreshParams()); err != nil {
		slog.Error("failed to check for protocol changes", "error", err.Error())
		return retry()
	}

	slog.Info("acquiring lock", "cycles", cycles, "phase", "acquiring_lock")
	unlock, err := lockCyclesWithTimeout(time.Minute*10, cycles...)
	if err != nil {
		slog.Error("failed to acquire lock", "error", err.Error())
		return retry()
	}
	defer unlock()

	slog.Info("===================== PROCESSING START =====================")
	slog.Info("processing cycles", "cycles", cycles)

	generationResult, err := generatePayoutsForCycles(cycles, config, collector, signer, &common.GeneratePayoutsOptions{})
	if err != nil {
		if errors.Is(err, constants.ErrNoCycleDataAvailable) {
			slog.Info("no data available for cycle, skipping", "cycle", cycleToProcess)
			return
		}
		slog.Error("failed to generate payouts", "error", err.Error())
		return retry()
	}

	slog.Info("checking reports of past payouts")
	preparationResult := assertRunWithResult(func() (*common.PreparePayoutsResult, error) {
		return core.PreparePayouts(generationResult, config, common.NewPreparePayoutsEngineContext(collector, signer, fsReporter, notifyAdminFactory(config)), &common.PreparePayoutsOptions{
			WaitForSufficientBalance: true,
			Accumulate:               true,
		})
	}, EXIT_OPERTION_FAILED)

	if len(preparationResult.ValidPayouts) == 0 {
		slog.Info("nothing to pay out, skipping")
		return
	}

	slog.Info("processing payouts", "valid", len(preparationResult.ValidPayouts), "invalid", len(preparationResult.InvalidPayouts), "accumulated", len(preparationResult.ValidPayouts), "already_successful", len(preparationResult.ReportsOfPastSuccessfulPayouts))

	if forceConfirmationPrompt && utils.IsTty() {
		utils.PrintPreparePayoutsResult(preparationResult, &utils.PrintPreparePayoutsResultOptions{AutoMergeRecords: true})
		msg := "Do you want to pay out above VALID payouts?"
		if isDryRun {
			msg = msg + " " + constants.DRY_RUN_NOTE
		}
		assertRequireConfirmation(msg)
	}

	slog.Info("executing payouts", "valid", len(preparationResult.ValidPayouts), "invalid", len(preparationResult.InvalidPayouts), "accumulated", len(preparationResult.ValidPayouts), "already_successful", len(preparationResult.ReportsOfPastSuccessfulPayouts))
	executionResult := assertRunWithResult(func() (*common.ExecutePayoutsResult, error) {
		return core.ExecutePayouts(preparationResult, config, common.NewExecutePayoutsEngineContext(signer, transactor, fsReporter, notifyAdminFactory(config)), &common.ExecutePayoutsOptions{
			MixInContractCalls: mixInContractCalls,
			MixInFATransfers:   mixInFATransfers,
			DryRun:             isDryRun,
		})
	}, EXIT_OPERTION_FAILED)

	// notify
	failedCount := lo.CountBy(executionResult.BatchResults, func(br common.BatchResult) bool { return !br.IsSuccess })
	if len(executionResult.BatchResults) > 0 {
		if failedCount > 0 {
			slog.Error("failed operations detected", "failed", failedCount, "total", len(executionResult.BatchResults), "cycle", cycleToProcess, "phase", "cycle_processing_failed")
			notifyAdmin(config, fmt.Sprintf("Failed operations detected: %d/%d in cycle %d", failedCount, len(executionResult.BatchResults), cycleToProcess))
			return
		} else {
			slog.Info("all operations succeeded", "total", len(executionResult.BatchResults), "cycle", cycleToProcess, "phase", "cycle_processing_success")
		}
	}
	if !silent && !isDryRun {
		notifyPayoutsProcessedThroughAllNotificators(config, &executionResult.Summary)
	}
	PrintPayoutWalletRemainingBalance(collector, signer)
	return
}

var continualCmd = &cobra.Command{
	Use:   "continual",
	Short: "continual payout",
	Long:  "runs payout until stopped manually",
	Run: func(cmd *cobra.Command, args []string) {
		configurationContext := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE)
		config, collector, _, _ := configurationContext.Unwrap()
		defer extension.CloseExtensions()
		initialCycle, _ := cmd.Flags().GetInt64(CYCLE_FLAG)
		endCycle, _ = cmd.Flags().GetInt64(END_CYCLE_FLAG)
		mixInContractCalls, _ := cmd.Flags().GetBool(DISABLE_SEPARATE_SC_PAYOUTS_FLAG)
		mixInFATransfers, _ := cmd.Flags().GetBool(DISABLE_SEPARATE_FA_PAYOUTS_FLAG)
		forceConfirmationPrompt, _ := cmd.Flags().GetBool(FORCE_CONFIRMATION_PROMPT_FLAG)
		isDryRun, _ := cmd.Flags().GetBool(DRY_RUN_FLAG)
		silent, _ := cmd.Flags().GetBool(SILENT_FLAG)
		payoutPeriod, _ := cmd.Flags().GetInt64(PAYOUT_PERIOD_FLAG)
		payoutPeriod = getBoundedPayoutPeriod(payoutPeriod)

		if isDryRun {
			slog.Info("Dry run mode enabled")
		}

		if forceConfirmationPrompt {
			if utils.IsTty() {
				slog.Info("you will be prompted for confirmation before each payout")
				time.Sleep(time.Second * 5)
			} else {
				slog.Error("force confirmation mode is not supported in non-interactive mode")
				os.Exit(EXIT_INVALID_ARGS)
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
		onchainCompletedCycle = assertRunWithResultAndErrorMessage(func() (int64, error) {
			return collector.GetLastCompletedCycle()
		}, EXIT_OPERTION_FAILED, "failed to get last completed cycle")

		lastProcessedCycle = onchainCompletedCycle
		if initialCycle != 0 {
			if initialCycle > 0 {
				lastProcessedCycle = initialCycle - 1
			} else {
				lastProcessedCycle = onchainCompletedCycle + initialCycle
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
				slog.Info("waiting for next cycle to complete", "phase", "waiting_for_next_cycle")
				var err error
				onchainCompletedCycle, err = monitor.WaitForNextCompletedCycle(lastProcessedCycle)
				if err != nil {
					if errors.Is(err, constants.ErrMonitoringCanceled) {
						slog.Info("cycle monitoring canceled", "phase", "cycle_monitoring_canceled")
						notifyAdmin(config, "Cycle monitoring canceled.")
					} else {
						slog.Error("failed to wait for next completed cycle", "error", err.Error(), "phase", "failed_to_wait_for_next_completed_cycle")
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
					slog.Warn("protocol changed, operator action required", "old_protocol", startupProtocol, "new_protocol", currentProtocol, "phase", "waiting_for_operator_action")
					notifyAdmin(config, fmt.Sprintf("Protocol changed from %s to %s, waiting for the operator to take action.", startupProtocol, currentProtocol))
					continue
				}
			}

			if !notifiedNewVersionAvailable {
				if available, latest := checkForNewVersionAvailable(); available {
					notifyAdmin(config, fmt.Sprintf("New tezpay version available - %s", latest))
					notifiedNewVersionAvailable = true
				}
			}

			processCycleInContinualMode(configurationContext, forceConfirmationPrompt, mixInContractCalls, mixInFATransfers, isDryRun, silent, payoutPeriod)
		}
	},
}

func init() {
	continualCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "initial cycle")
	continualCmd.Flags().Int64P(END_CYCLE_FLAG, "e", 0, "end cycle")
	continualCmd.Flags().Int64(PAYOUT_PERIOD_FLAG, 1, "payout period")
	continualCmd.Flags().Bool(DISABLE_SEPARATE_SC_PAYOUTS_FLAG, false, "disables smart contract separation (mixes txs and smart contract calls within batches)")
	continualCmd.Flags().Bool(DISABLE_SEPARATE_FA_PAYOUTS_FLAG, false, "disables fa transfers separation (mixes txs and fa transfers within batches)")
	continualCmd.Flags().BoolP(FORCE_CONFIRMATION_PROMPT_FLAG, "a", false, "ask for confirmation on each payout")
	continualCmd.Flags().Bool(DRY_RUN_FLAG, false, "Performs all actions except sending transactions. Reports are stored in 'reports/dry' folder")

	RootCmd.AddCommand(continualCmd)
}
