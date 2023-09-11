package utils

import (
	"context"
	"encoding/json"
	"fmt"
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
	fmt.Printf("%s - metadata %s:\n", reason, id)
	for _, m := range metadata {
		data, err := json.Marshal(m)
		if err == nil {
			fmt.Println(string(data))
			continue
		}
		fmt.Printf("Failed to marshal metadata: %s\n", err)
	}
	fmt.Printf("Please report above metadata to the developers.\n")
	panic(reason)
}
