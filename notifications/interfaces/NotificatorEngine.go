package interfaces

import "github.com/alis-is/tezpay/core/payout/common"

type NotificatorEngine interface {
	Notify(summary *common.CyclePayoutSummary) error
	TestNotify() error
}
