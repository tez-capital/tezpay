package utils

import (
	"context"
	"os"
	"os/signal"
)

func CallbackOnInterrupt(ctx context.Context, cb func()) {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		select {
		case <-c:
			cb()
		case <-ctx.Done():
		}
		signal.Reset(os.Interrupt)
	}()
}
