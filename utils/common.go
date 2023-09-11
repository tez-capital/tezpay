package utils

import (
	"context"
	"time"
)

type ICliPrintable interface {
	GetHeaders() []string
	ToTableRow() []string
}

func SleepContext(ctx context.Context, delay time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}
