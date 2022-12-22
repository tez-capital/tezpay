package cmd

import (
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/notifications"
	log "github.com/sirupsen/logrus"
)

func notifyPayoutsProcessed(configuration *configuration.RuntimeConfiguration, summary *common.CyclePayoutSummary, filter string) {
	for _, notificatorConfiguration := range configuration.NotificationConfigurations {
		if filter != "" && string(notificatorConfiguration.Type) != filter {
			continue
		}

		if notificatorConfiguration.IsAdmin {
			continue
		}

		log.Infof("sending notification with %s", notificatorConfiguration.Type)
		notificator, err := notifications.LoadNotificatior(notificatorConfiguration.Type, notificatorConfiguration.Configuration)
		if err != nil {
			log.Warnf("failed to send notification - %s", err.Error())
			continue
		}

		err = notificator.PayoutSummaryNotify(summary)
		if err != nil {
			log.Warnf("failed to send notification - %s", err.Error())
			continue
		}
	}
	log.Info("notifications sent.")
}
func notifyPayoutsProcessedThroughAllNotificators(configuration *configuration.RuntimeConfiguration, summary *common.CyclePayoutSummary) {
	notifyPayoutsProcessed(configuration, summary, "")
}

func notifyAdmin(configuration *configuration.RuntimeConfiguration, msg string) {
	for _, notificatorConfiguration := range configuration.NotificationConfigurations {
		if !notificatorConfiguration.IsAdmin {
			continue
		}

		log.Infof("sending admin notification with %s", notificatorConfiguration.Type)
		notificator, err := notifications.LoadNotificatior(notificatorConfiguration.Type, notificatorConfiguration.Configuration)
		if err != nil {
			log.Warnf("failed to send notification - %s", err.Error())
			continue
		}

		err = notificator.AdminNotify(msg)
		if err != nil {
			log.Warnf("failed to send notification - %s", err.Error())
			continue
		}
	}
	log.Info("admin notifications sent.")
}

func notifyAdminFactory(configuration *configuration.RuntimeConfiguration) func(string) {
	return func(msg string) {
		notifyAdmin(configuration, msg)
	}
}
