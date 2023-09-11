//go:build !wasm

package main

import (
	"os"

	"github.com/alis-is/tezpay/cmd"
	"github.com/alis-is/tezpay/common"
	log "github.com/sirupsen/logrus"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			if panicStatus, ok := r.(common.PanicStatus); ok {
				os.Exit(panicStatus.ExitCode)
			} else {
				log.Fatal("Unhandled panic")
			}
		}
	}()

	cmd.Execute()
}
