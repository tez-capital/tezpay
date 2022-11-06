package utils

import (
	"os"
	"os/signal"
)

func CallbackOnInterrupt(cb func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cb()
	}()
}
