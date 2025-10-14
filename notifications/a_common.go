package notifications

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/trilitech/tzgo/tezos"
)

type NotificationKind string

const (
	PAYOUT_SUMMARY_NOTIFICATION NotificationKind = "payout_summary"
	ADMIN_NOTIFICATION          NotificationKind = "admin"
	TEST_NOTIFICATION           NotificationKind = "test"
	TEXT_NOTIFICATION           NotificationKind = "text"
)

type NotificatorKind string

const (
	TELEGRAM_NOTIFICATOR NotificatorKind = "telegram"
	TWITTER_NOTIFICATOR  NotificatorKind = "twitter"
	DISCORD_NOTIFICATOR  NotificatorKind = "discord"
	EMAIL_NOTIFICATOR    NotificatorKind = "email"
	EXTERNAL_NOTIFICATOR NotificatorKind = "external"
	WEBHOOK_NOTIFICATOR  NotificatorKind = "webhook"
	BLUESKY_NOTIFICATOR  NotificatorKind = "bluesky"
)

func PopulateMessageTemplate(messageTempalte string, summary *common.PayoutSummary, additionalData map[string]string) string {
	v := reflect.ValueOf(*summary)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		val := fmt.Sprintf("%v", v.Field(i).Interface())
		if typeOfS.Field(i).Type.Name() == "Z" && strings.Contains(typeOfS.Field(i).Type.PkgPath(), "tzgo/tezos") {
			val = fmt.Sprintf("%v", common.MutezToTezS(v.Field(i).Interface().(tezos.Z).Int64()))
		}
		if typeOfS.Field(i).Name == "Cycles" {
			val = strings.Join(lo.Map(summary.Cycles, func(c int64, _ int) string {
				return fmt.Sprintf("#%d", c)
			}), ", ")
		}
		messageTempalte = strings.ReplaceAll(messageTempalte, fmt.Sprintf("<%s>", typeOfS.Field(i).Name), val)
		if typeOfS.Field(i).Name == "Cycles" { // backward compatibility
			messageTempalte = strings.ReplaceAll(messageTempalte, "<Cycle>", val)
		}
	}

	for k, v := range additionalData {
		messageTempalte = strings.ReplaceAll(messageTempalte, fmt.Sprintf("<%s>", k), v)
	}

	return messageTempalte
}
