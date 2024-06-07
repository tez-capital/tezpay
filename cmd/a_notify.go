package cmd

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/extension"
	"github.com/tez-capital/tezpay/notifications"
)

func collectAdditionalData(_ *common.CyclePayoutSummary) map[string]string {
	data := make(map[string]json.RawMessage)

	err := extension.ExecuteHook(enums.EXTENSION_HOOK_COLLECT_ADDITIONAL_NOTIFICATION_DATA, "0.1", &data)
	if err != nil {
		log.Warnf("failed to execute hook - %s", err.Error())
	}
	result := make(map[string]string)
	for key, value := range data {
		result[key] = string(value)
	}

	return result
}

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

		additionalData := collectAdditionalData(summary)
		err = notificator.PayoutSummaryNotify(summary, additionalData)
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

		log.Debugf("sending admin notification with %s", notificatorConfiguration.Type)
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
	log.Debug("admin notifications sent.")
}

func notifyAdminFactory(configuration *configuration.RuntimeConfiguration) func(string) {
	return func(msg string) {
		notifyAdmin(configuration, msg)
	}
}
