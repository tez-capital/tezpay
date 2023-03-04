package cmd

import (
	"fmt"
	"os"

	"github.com/alis-is/tezpay/configuration/seed"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/state"
	"github.com/echa/log"
	"github.com/spf13/cobra"
)

var generateConfigurationCmd = &cobra.Command{
	Use:     "import-configuration <kind> <source-file>",
	Short:   "seed configuration from",
	Aliases: []string{"import-config"},
	Long:    "generates configuration based on configuration from others payout distribution tools",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
			return err
		}

		seedKind := enums.EConfigurationSeedKind(args[0])
		switch seedKind {
		case enums.BC_CONFIGURATION_SEED:
		case enums.TRD_CONFIGURATION_SEED:
		default:
			return fmt.Errorf("invalid seed kind: %s", seedKind)
		}
		if _, err := os.Stat(args[1]); err != nil {
			return fmt.Errorf("invalid source file: %s", args[1])
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		sourceFile := args[1]
		destiantionFile := state.Global.GetConfigurationFilePath()

		if _, err := os.Stat(destiantionFile); err == nil {
			assertRequireConfirmation("configuration file already exists, overwrite?")
		}

		// load source bytes
		sourceBytes := assertRunWithResultAndErrFmt(func() ([]byte, error) {
			return os.ReadFile(sourceFile)
		}, EXIT_CONFIGURATION_LOAD_FAILURE, "failed to read source file - %s")

		seededBytes, err := seed.Generate(sourceBytes, enums.EConfigurationSeedKind(args[0]))
		if err != nil {
			log.Errorf("failed to generate configuration - %s", err)
			os.Exit(EXIT_CONFIGURATION_GENERATE_FAILURE)
		}
		assertRunWithErrFmt(func() error {
			if target, err := os.Stat(destiantionFile); err == nil {
				if source, err := os.Stat(sourceFile); err == nil {
					if os.SameFile(target, source) {
						// backup old configuration file
						return os.Rename(destiantionFile, destiantionFile+constants.CONFIG_FILE_BACKUP_SUFFIX)
					}
				}
			}
			return os.WriteFile(destiantionFile, seededBytes, 0644)
		}, EXIT_CONFIGURATION_SAVE_FAILURE, "failed to save configuration file - %s")
		log.Info("tezpay configuration generated successfully")
	},
}

func init() {
	RootCmd.AddCommand(generateConfigurationCmd)
}
