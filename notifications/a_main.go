package notifications

import (
	"fmt"

	"github.com/alis-is/tezpay/notifications/interfaces"
)

func LoadNotificatior(kind string, configuration []byte) (interfaces.NotificatorEngine, error) {
	switch kind {
	case "twitter":
		return InitTwitterNotificator(configuration)
	case "discord":
		return InitDiscordNotificator(configuration)
	default:
		return nil, fmt.Errorf("not supported plugin %s", kind)
	}
}

func ValidateNotificatorConfiguration(kind string, configuration []byte) error {
	switch kind {
	case "twitter":
		return ValidateTwitterConfiguration(configuration)
	case "discord":
		return ValidateDiscordConfiguration(configuration)
	default:
		return fmt.Errorf("not supported plugin %s", kind)
	}
}
