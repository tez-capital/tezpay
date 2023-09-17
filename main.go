//go:build !wasm

package main

import (
	"fmt"
	"os"

	"github.com/alis-is/tezpay/cmd"
	"github.com/alis-is/tezpay/common"
	log "github.com/sirupsen/logrus"
)

func main() {
	defer func() {
		if !containsDebugFlag(os.Args) {
			if r := recover(); r != nil {
				if panicStatus, ok := r.(common.PanicStatus); ok {
					os.Exit(panicStatus.ExitCode)
				} else {
					fmt.Println(r)
					log.Fatal("Unhandled panic")
				}
			}
		}
	}()

	cmd.Execute()
}

func containsDebugFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--"+cmd.DEBUG_FLAG {
			return true
		}
	}
	return false
}
