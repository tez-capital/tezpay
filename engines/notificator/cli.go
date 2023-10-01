//go:build !wasm

package notificator_engines

import (
	"fmt"

	"github.com/alis-is/tezpay/common"
)

func LoadNotificators(kind string, configuration []byte) (common.NotificatorEngine, error) {
	switch NotificatorKind(kind) {
	case TWITTER_NOTIFICATOR:
		return InitTwitterNotificator(configuration)
	case DISCORD_NOTIFICATOR:
		return InitDiscordNotificator(configuration)
	case TELEGRAM_NOTIFICATOR:
		return InitTelegramNotificator(configuration)
	case EMAIL_NOTIFICATOR:
		return InitEmailNotificator(configuration)
	case EXTERNAL_NOTIFICATOR:
		return InitExternalNotificator(configuration)
	default:
		return nil, fmt.Errorf("not supported notificator %s", kind)
	}
}

func ValidateNotificatorConfiguration(kind string, configuration []byte) error {
	switch NotificatorKind(kind) {
	case TWITTER_NOTIFICATOR:
		return ValidateTwitterConfiguration(configuration)
	case DISCORD_NOTIFICATOR:
		return ValidateDiscordConfiguration(configuration)
	case TELEGRAM_NOTIFICATOR:
		return ValidateTelegramConfiguration(configuration)
	case EMAIL_NOTIFICATOR:
		return ValidateEmailConfiguration(configuration)
	case EXTERNAL_NOTIFICATOR:
		return ValidateExternalConfiguration(configuration)
	default:
		return fmt.Errorf("not supported notificator %s", kind)
	}
}
