package cmd

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core"
	reporter_engines "github.com/alis-is/tezpay/engines/reporter"
	"github.com/alis-is/tezpay/extension"
	"github.com/alis-is/tezpay/state"
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
		config, collector, signer, transactor := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		defer extension.CloseExtensions()
		initialCycle, _ := cmd.Flags().GetInt64(CYCLE_FLAG)
		endCycle, _ := cmd.Flags().GetInt64(END_CYCLE_FLAG)
		mixInContractCalls, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG)
		mixInFATransfers, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_FA_PAYOUTS_FLAG)
		forceConfirmationPrompt, _ := cmd.Flags().GetBool(FORCE_CONFIRMATION_PROMPT_FLAG)

		fsReporter := reporter_engines.NewFileSystemReporter(config)

		if utils.IsTty() {
			assertRequireConfirmation("\n\n\t !!! ATTENTION !!!\n\nPreliminary testing has been conducted on the continual mode, but potential for undiscovered bugs still exists.\n Do you want to proceed?")
		}
		if forceConfirmationPrompt {
			if utils.IsTty() {
				log.Info("you will be prompted for confirmation before each payout")
			} else {
				log.Warn("force confirmation mode is not supported in non-interactive mode")
			}
		}

		if !state.Global.IsDonationPromptDisabled() && !config.IsDonatingToTezCapital() {
			assertRequireConfirmation("With your current configuration you are not going to donate to tez.capital. Do you want to proceed?")
		}

		monitor := assertRunWithResultAndErrFmt(func() (common.CycleMonitor, error) {
			return collector.CreateCycleMonitor(common.CycleMonitorOptions{
				CheckFrequency:    10,
				NotificationDelay: rand.Int63n(config.PayoutConfiguration.MaximumDelayBlocks-config.PayoutConfiguration.MinimumDelayBlocks) + config.PayoutConfiguration.MinimumDelayBlocks,
			})
		}, EXIT_OPERTION_FAILED, "failed to init cycle monitor")

		// last completed cycle at the time we started continual mode on
		onchainCompletedCycle := assertRunWithResultAndErrFmt(func() (int64, error) {
			return collector.GetLastCompletedCycle()
		}, EXIT_OPERTION_FAILED, "failed to get last completed cycle")

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
			log.Infof("================  CYCLE %d PROCESSED ===============", cycleToProcess)
			if endCycle != 0 && lastProcessedCycle >= endCycle {
				log.Info("end cycle reached, exiting")
				os.Exit(0)
			}
		}

		notifiedNewVersionAvailable := false

		startupProtocol := GetProtocolWithRetry(collector)
		if !config.Network.IgnoreProtocolChanges {
			log.Infof("Continual mode started in safe mode. In the event of a protocol change, TezPay will stop processing payouts and you will be notified.")
		}
		defer func() {
			notifyAdmin(config, fmt.Sprintf("Continual payouts stopped on cycle #%d", lastProcessedCycle+1))
		}()
		notifyAdmin(config, fmt.Sprintf("Continual payouts started on cycle #%d (tezpay %s, protocol %s)", lastProcessedCycle+1, constants.VERSION, startupProtocol))
		for {
			if lastProcessedCycle >= onchainCompletedCycle {
				log.Info("looking for next cycle to pay out")
				var err error
				onchainCompletedCycle, err = monitor.WaitForNextCompletedCycle(lastProcessedCycle)
				if err != nil {
					if errors.Is(err, constants.ErrMonitoringCanceled) {
						log.Info("cycle monitoring canceled")
						notifyAdmin(config, "Cycle monitoring canceled.")
					} else {
						log.Errorf("failed to wait for next completed cycle - %s", err.Error())
						notifyAdmin(config, "Failed to wait for next completed cycle.")
					}
					return
				}
			}
			if !config.Network.IgnoreProtocolChanges {
				log.Debugf("Checking current protocol...")
				currentProtocol := GetProtocolWithRetry(collector)
				if currentProtocol != startupProtocol {
					/// we can not exit here. Users may configure recover mechanism in case of crashes/exits so we really want to wait for the operator to take action
					log.Errorf("Protocol changed from %s to %s, waiting for the operator to take action.", startupProtocol, currentProtocol)
					notifyAdmin(config, fmt.Sprintf("Protocol changed from %s to %s, waiting for the operator to take action.", startupProtocol, currentProtocol))
					continue
				}
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
				log.Errorf("failed to refresh chain params - %s, retries in 5 minutes", err.Error())
				time.Sleep(time.Minute * 5)
				continue
			}

			log.Infof("====================  CYCLE %d  ====================", cycleToProcess)

			generationResult, err := core.GeneratePayouts(config, common.NewGeneratePayoutsEngines(collector, signer, notifyAdminFactory(config)),
				&common.GeneratePayoutsOptions{
					Cycle:                    cycleToProcess,
					WaitForSufficientBalance: true,
				})
			if err != nil {
				if errors.Is(err, constants.ErrNoCycleDataAvailable) {
					log.Infof("no data available for cycle %d, skipping...", cycleToProcess)
					completeCycle()
					continue
				}
				log.Errorf("failed to generate payout - %s, retries in 5 minutes", err.Error())
				time.Sleep(time.Minute * 5)
				continue
			}

			log.Info("checking past reports")
			preparationResult := assertRunWithResult(func() (*common.PreparePayoutsResult, error) {
				return core.PrepareCyclePayouts(generationResult, config, common.NewPreparePayoutsEngineContext(collector, fsReporter, notifyAdminFactory(config)), &common.PreparePayoutsOptions{})
			}, EXIT_OPERTION_FAILED)

			if len(preparationResult.ValidPayouts) == 0 {
				log.Info("nothing to pay out, skipping")
				completeCycle()
				continue
			}
			log.Infof("processing %d valid payouts", len(preparationResult.ValidPayouts))

			if forceConfirmationPrompt && utils.IsTty() {
				cycles := []int64{generationResult.Cycle}
				utils.PrintInvalidPayoutRecipes(preparationResult.ValidPayouts, cycles)
				utils.PrintReports(preparationResult.ReportsOfPastSuccesfulPayouts, fmt.Sprintf("Already Successfull - %s", utils.FormatCycleNumbers(cycles)), true)
				utils.PrintValidPayoutRecipes(preparationResult.ValidPayouts, cycles)
				assertRequireConfirmation("Do you want to pay out above VALID payouts?")
			}

			log.Info("executing payout")
			executionResult := assertRunWithResult(func() (*common.ExecutePayoutsResult, error) {
				return core.ExecutePayouts(preparationResult, config, common.NewExecutePayoutsEngineContext(signer, transactor, fsReporter, notifyAdminFactory(config)), &common.ExecutePayoutsOptions{
					MixInContractCalls: mixInContractCalls,
					MixInFATransfers:   mixInFATransfers,
				})
			}, EXIT_OPERTION_FAILED)

			// notify
			failedCount := lo.CountBy(executionResult.BatchResults, func(br common.BatchResult) bool { return !br.IsSuccess })
			if len(executionResult.BatchResults) > 0 && failedCount > 0 {
				log.Errorf("%d of operations failed, retries in 5 minutes", failedCount)
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

	RootCmd.AddCommand(continualCmd)
}
