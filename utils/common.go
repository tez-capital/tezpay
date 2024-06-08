package utils

import (
	"context"
	"log/slog"
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

func PanicWithMetadata(reason string, id string, metadata ...interface{}) {
	slog.Error(reason, "id", id, "metadata", metadata)
	slog.Info("Please report above to the developers.")
	panic(reason)
}

func MapToPointers[T any](items []T) []*T {
	pointers := make([]*T, len(items))
	for i, item := range items {
		pointers[i] = &item
	}
	return pointers
}
