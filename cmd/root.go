package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	signer_engines "github.com/tez-capital/tezpay/engines/signer"

	"log/slog"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezpay/state"
	"github.com/tez-capital/tezpay/utils"
)

const (
	LOG_LEVEL_FLAG               = "log-level"
	PATH_FLAG                    = "path"
	DISABLE_DONATION_PROMPT_FLAG = "disable-donation-prompt"
	OUTPUT_FORMAT_FLAG           = "output-format"
	PAY_ONLY_ADDRESS_PREFIX      = "pay-only-address-prefix"
)

var (
	LOG_LEVEL_MAP = map[string]slog.Level{
		"":      slog.LevelInfo,
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
)

func setupJsonLogger(level slog.Level) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

func setupTextLogger(level slog.Level) {
	handler := utils.NewPrettyTextLogHandler(os.Stdout, utils.PrettyHandlerOptions{
		HandlerOptions: slog.HandlerOptions{Level: level},
	})
	slog.SetDefault(slog.New(handler))
}

var (
	RootCmd = &cobra.Command{
		Use:   "tezpay",
		Short: "TEZPAY",
		Long: fmt.Sprintf(`TEZPAY %s - the tezos reward distributor
Copyright © %d alis.is
`, constants.VERSION, time.Now().Year()),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			format, _ := cmd.Flags().GetString(OUTPUT_FORMAT_FLAG)
			disableDonationPrompt, _ := cmd.Flags().GetBool(DISABLE_DONATION_PROMPT_FLAG)
			level, _ := cmd.Flags().GetString(LOG_LEVEL_FLAG)

			switch format {
			case "json":
				setupJsonLogger(LOG_LEVEL_MAP[level])
			case "text":
				setupTextLogger(LOG_LEVEL_MAP[level])
			default:
				if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
					setupJsonLogger(LOG_LEVEL_MAP[level])
				} else {
					setupTextLogger(LOG_LEVEL_MAP[level])
				}
			}
			slog.Debug("logger configured", "format", format, "level", level)

			workingDirectory, _ := cmd.Flags().GetString(PATH_FLAG)
			singerFlagData, _ := cmd.Flags().GetString(SIGNER_FLAG)
			var signerOverride common.SignerEngine
			if singerFlagData != "" {
				slog.Debug("trying to load signer override")
				if loadedSigner, err := signer_engines.Load(singerFlagData); err != nil {
					slog.Warn("failed to load signer from parameters", "error", err)
				} else {
					signerOverride = loadedSigner
				}
			}

			payOnlyAddressPrefix, _ := cmd.Flags().GetString(PAY_ONLY_ADDRESS_PREFIX)
			if payOnlyAddressPrefix != "" {
				slog.Warn("Paying out only addresses starting with specified prefix", "prefix", payOnlyAddressPrefix)
			}

			stateOptions := state.StateInitOptions{
				WantsJsonOutput:       format == "json",
				SignerOverride:        signerOverride,
				Debug:                 level == "trace" || level == "debug",
				DisableDonationPrompt: disableDonationPrompt,
				PayOnlyAddressPrefix:  payOnlyAddressPrefix,
			}
			if err := state.Init(workingDirectory, stateOptions); err != nil {
				slog.Error("Failed to initialize state", "error", err)
				os.Exit(EXIT_STATE_LOAD_FAILURE)
			}

			skipVersionCheck, _ := cmd.Flags().GetBool(SKIP_VERSION_CHECK_FLAG)
			if !skipVersionCheck && utils.IsTty() {
				promptIfNewVersionAvailable()
			}
		},
	}
)

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.PersistentFlags().StringP(PATH_FLAG, "p", ".", "path to working directory")
	RootCmd.PersistentFlags().StringP(OUTPUT_FORMAT_FLAG, "o", "auto", "Sets output log format (json/text/auto)")
	RootCmd.PersistentFlags().StringP(LOG_LEVEL_FLAG, "l", "info", "Sets log level format (trace/debug/info/warn/error)")
	RootCmd.PersistentFlags().String(SIGNER_FLAG, "", "Override signer")
	RootCmd.PersistentFlags().Bool(SKIP_VERSION_CHECK_FLAG, false, "Skip version check")
	RootCmd.PersistentFlags().Bool(DISABLE_DONATION_PROMPT_FLAG, false, "Disable donation prompt")
	RootCmd.PersistentFlags().String(PAY_ONLY_ADDRESS_PREFIX, "", "Pays only to addresses starting with the prefix (e.g. KT, usually you do not want to use this, just for recovering in case of issues)")
	RootCmd.PersistentFlags().SetInterspersed(false)
}
