package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/extension"
)

type testHookData struct {
	Message string `json:"message"`
}

var extensionTestCmd = &cobra.Command{
	Use:   "test-extensions",
	Short: "extensions test",
	Long:  "initializes and executes test hook agains extensions",
	Run: func(cmd *cobra.Command, args []string) {
		assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		defer extension.CloseScopedExtensions()

		data := testHookData{
			Message: "hello",
		}
		if err := extension.ExecuteHook(enums.EXTENSION_HOOK_TEST_NOTIFY, "0.1", &data); err != nil {
			slog.Error("failed to execute hook", "error", err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
			return
		}
		slog.Info("test-notify hook executed successfully")
		extension.CloseScopedExtensions()
		if err := extension.ExecuteHook(enums.EXTENSION_HOOK_TEST_REQUEST, "0.1", &data); err != nil {
			slog.Error("failed to execute hook", "error", err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
			return
		}
		slog.Info("test-request hook executed successfully", "response message", data.Message)
	},
}

func init() {
	// TODO: supply test data file
	RootCmd.AddCommand(extensionTestCmd)
}
