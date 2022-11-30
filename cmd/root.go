package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/signer"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	LOG_LEVEL_FLAG     = "log-level"
	PATH_FLAG          = "path"
	OUTPUT_FORMAT_FLAG = "output-format"
)

var (
	LOG_LEVEL_MAP = map[string]log.Level{
		"":      log.InfoLevel,
		"trace": log.TraceLevel,
		"debug": log.DebugLevel,
		"info":  log.InfoLevel,
		"warn":  log.WarnLevel,
		"error": log.ErrorLevel,
	}
)
var (
	RootCmd = &cobra.Command{
		Use:   "tezpay",
		Short: "TEZPAY",
		Long: fmt.Sprintf(`TEZPAY %s - the tezos reward distributor
Copyright © %d alis.is
`, constants.VERSION, time.Now().Year()),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			format, _ := cmd.Flags().GetString(OUTPUT_FORMAT_FLAG)
			outputJson := false

			switch format {
			case "json":
				outputJson = true
				log.SetFormatter(&utils.LogJsonFormatter{})
				log.Trace("Output format set to 'json'")
			case "text":
				log.SetFormatter(&utils.LogTextFormatter{})
				log.Trace("Output format set to 'text'")
			default:
				if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
					outputJson = true
					log.SetFormatter(&utils.LogJsonFormatter{})
					log.Trace("Output format automatically set to 'json'")
				} else {
					log.SetFormatter(&utils.LogTextFormatter{})
					log.Trace("Output format automatically set to 'text'")
				}
			}

			workingDirectory, _ := cmd.Flags().GetString(PATH_FLAG)

			level, _ := cmd.Flags().GetString("log-level")
			log.SetLevel(LOG_LEVEL_MAP[level])
			log.Trace("Log level set to '" + log.GetLevel().String() + "'")

			singerFlagData, _ := cmd.Flags().GetString(SIGNER_FLAG)
			var signerOverride common.SignerEngine
			if singerFlagData != "" {
				log.Debug("trying to load signer override")
				if loadedSigner, err := signer.Load(singerFlagData); err != nil {
					log.Warnf("Failed to load signer from parameters (%s)", singerFlagData)
				} else {
					signerOverride = loadedSigner
				}
			}

			state.Init(workingDirectory, state.StateInitOptions{
				WantsJsonOutput: outputJson,
				SignerOverride:  signerOverride,
				Debug:           level == "trace" || level == "debug",
			})

			skipVersionCheck, _ := cmd.Flags().GetBool(SKIP_VERSION_CHECK_FLAG)
			if !skipVersionCheck {
				checkLatestVersion()
			}
		},
	}
)

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.PersistentFlags().StringP(PATH_FLAG, "p", ".", "Path to bake buddy instance")
	RootCmd.PersistentFlags().StringP(OUTPUT_FORMAT_FLAG, "o", "auto", "Sets log level format (trace/debug/info/warn/error)")
	RootCmd.PersistentFlags().StringP(LOG_LEVEL_FLAG, "l", "info", "Sets output log format (json/text/auto)")
	RootCmd.PersistentFlags().String(SIGNER_FLAG, "", "Override signer")
	RootCmd.PersistentFlags().Bool(SKIP_VERSION_CHECK_FLAG, false, "Skip version check")
	RootCmd.PersistentFlags().SetInterspersed(false)
}
