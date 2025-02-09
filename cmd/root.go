package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	signer_engines "github.com/tez-capital/tezpay/engines/signer"

	"log/slog"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezpay/state"
	"github.com/tez-capital/tezpay/utils"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LOG_LEVEL_FLAG               = "log-level"
	LOG_SERVER_FLAG              = "log-server"
	LOG_FILE_FLAG                = "log-file"
	PATH_FLAG                    = "path"
	VERSION_FLAG                 = "version"
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

func setupLumberjackLogger(logFile string) io.Writer {
	return &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	}
}

func setupJsonLogger(level slog.Level, logServerAddress string, logFile string) {
	writers := make([]io.Writer, 0, 3)
	writers = append(writers, os.Stdout)

	if logServerAddress != "" {
		writers = append(writers, utils.NewLogServer(logServerAddress))
	}
	if logFile != "" {
		writers = append(writers, setupLumberjackLogger(logFile))
	}

	handler := slog.NewJSONHandler(utils.NewMultiWriter(writers...), &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
	if logServerAddress != "" {
		slog.Info("log server started", "address", logServerAddress)
	}
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
			logServer, _ := cmd.Flags().GetString(LOG_SERVER_FLAG)
			logFile, _ := cmd.Flags().GetString(LOG_FILE_FLAG)

			switch format {
			case "json":
				setupJsonLogger(LOG_LEVEL_MAP[level], logServer, logFile)
			case "text":
				setupTextLogger(LOG_LEVEL_MAP[level])
			default:
				if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
					setupJsonLogger(LOG_LEVEL_MAP[level], logServer, logFile)
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
					slog.Warn("failed to load signer from parameters", "error", err.Error())
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
				DisableDonationPrompt: disableDonationPrompt,
				PayOnlyAddressPrefix:  payOnlyAddressPrefix,
			}
			if err := state.Init(workingDirectory, stateOptions); err != nil {
				slog.Error("Failed to initialize state", "error", err.Error())
				os.Exit(EXIT_STATE_LOAD_FAILURE)
			}

			skipVersionCheck, _ := cmd.Flags().GetBool(SKIP_VERSION_CHECK_FLAG)
			if !skipVersionCheck && utils.IsTty() {
				promptIfNewVersionAvailable()
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetBool(VERSION_FLAG)
			if version {
				fmt.Println(constants.VERSION)
				return
			}

			cmd.Help()
		},
	}
)

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.Flags().Bool(VERSION_FLAG, false, "Prints version")
	RootCmd.PersistentFlags().StringP(PATH_FLAG, "p", ".", "path to working directory")
	RootCmd.PersistentFlags().StringP(OUTPUT_FORMAT_FLAG, "o", "auto", "Sets output log format (json/text/auto)")
	RootCmd.PersistentFlags().StringP(LOG_LEVEL_FLAG, "l", "info", "Sets log level format (trace/debug/info/warn/error)")
	RootCmd.PersistentFlags().String(LOG_SERVER_FLAG, "", "launches log server at specified address")
	RootCmd.PersistentFlags().String(LOG_FILE_FLAG, "", "Logs to file")
	RootCmd.PersistentFlags().String(SIGNER_FLAG, "", "Override signer")
	RootCmd.PersistentFlags().Bool(SKIP_VERSION_CHECK_FLAG, false, "Skip version check")
	RootCmd.PersistentFlags().Bool(DISABLE_DONATION_PROMPT_FLAG, false, "Disable donation prompt")
	RootCmd.PersistentFlags().String(PAY_ONLY_ADDRESS_PREFIX, "", "Pays only to addresses starting with the prefix (e.g. KT, usually you do not want to use this, just for recovering in case of issues)")
	RootCmd.PersistentFlags().SetInterspersed(false)
}
