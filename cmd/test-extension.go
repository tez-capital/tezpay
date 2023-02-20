package cmd

import (
	"context"

	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/extension"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type testHookData struct {
	Message string `json:"message"`
}

var extensionTestCmd = &cobra.Command{
	Use:   "test-extension",
	Short: "extension test",
	Long:  "initializes and executes test hook agains extensions",
	Run: func(cmd *cobra.Command, args []string) {
		config, _, _, _ := assertRunWithResult(loadConfigurationAndEngines, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()

		err := extension.InitializeExtensionStore(context.Background(), config.Extensions)
		defer extension.EndExecutionScope()
		if err != nil {
			log.Errorf("failed to initialize extension store - %s", err.Error())
		}
		data := testHookData{
			Message: "hello",
		}
		if err := extension.ExecuteHook(enums.EXTENSION_HOOK_TEST_NOTIFY, "0.1", &data); err != nil {
			log.Errorf("failed to execute hook - %s", err.Error())
			return
		}
		log.Info("test-notify hook executed successfully")
		extension.EndExecutionScope()
		if err := extension.ExecuteHook(enums.EXTENSION_HOOK_TEST_REQUEST, "0.1", &data); err != nil {
			log.Errorf("failed to execute hook - %s", err.Error())
			return
		}
		log.Infof("test-request hook executed successfully - response message: %s", data.Message)
	},
}

func init() {
	// TODO: supply test data file
	RootCmd.AddCommand(extensionTestCmd)
}
