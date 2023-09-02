//go:build !wasm

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/alis-is/tezpay/configuration/seed"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/state"
	"github.com/echa/log"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

var generateConfigurationCmd = &cobra.Command{
	Use:     "import-configuration <kind> <source-file>",
	Short:   "seed configuration from",
	Aliases: []string{"import-config"},
	Long: `Generates configuration based on configuration from others payout distribution tools.

	Currently supported sources are: ` + strings.Join(lo.Map(enums.SUPPORTED_CONFIGURATION_SEED_KINDS, func(item enums.EConfigurationSeedKind, _ int) string {
		return string(item)
	}), ", ") + `

	To import configuration from supported sources copy configuration file to directory where you plan to store tezpay configuration and run command with source file path as argument.

	Example:
		tezpay import-configuration bc ./bc.json
		tezpay import-configuration trd ./trd.yaml
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
			return err
		}

		seedKind := enums.EConfigurationSeedKind(args[0])
		if !slices.Contains(enums.SUPPORTED_CONFIGURATION_SEED_KINDS, seedKind) {
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
			panic(PanicStatus{
				ExitCode: EXIT_CONFIGURATION_GENERATE_FAILURE,
				Error:    fmt.Errorf("failed to generate configuration - %s", err),
			})
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
