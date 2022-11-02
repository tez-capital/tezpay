package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/alis-is/tezpay/constants"
)

func MutezToTezS(amount int64) string {
	if amount == 0 {
		return "-"
	}
	tez := float64(amount) / constants.MUTEZ_FACTOR
	return fmt.Sprintf("%f TEZ", tez)
}

func SleepContext(ctx context.Context, delay time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}
