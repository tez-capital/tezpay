package interfaces

import "github.com/alis-is/tezpay/core/common"

type NotificatorEngine interface {
	Notify(summary *common.CyclePayoutSummary) error
	TestNotify() error
}
