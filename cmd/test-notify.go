package cmd

import (
	"github.com/alis-is/tezpay/notifications"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var notificationTestCmd = &cobra.Command{
	Use:   "test-notify",
	Short: "notification test",
	Long:  "sends test notification",
	Run: func(cmd *cobra.Command, args []string) {
		config, _, _, _ := assertRunWithResult(loadConfigurationAndEngines, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		notificator, _ := cmd.Flags().GetString(NOTIFICATOR_FLAG)
		for _, notificatorConfiguration := range config.NotificationConfigurations {
			if notificator != "" && notificatorConfiguration.Type != notificator {
				continue
			}

			log.Infof("Sending notification with %s", notificatorConfiguration.Type)
			notificator, err := notifications.LoadNotificatior(notificatorConfiguration.Type, notificatorConfiguration.Configuration)
			if err != nil {
				log.Warnf("failed to send notification - %s", err.Error())
				continue
			}

			err = notificator.TestNotify()
			if err != nil {
				log.Warnf("failed to send notification - %s", err.Error())
				continue
			}
		}
	},
}

func init() {
	notificationTestCmd.Flags().String(NOTIFICATOR_FLAG, "", "Notify through specific notificator")

	RootCmd.AddCommand(notificationTestCmd)
}
