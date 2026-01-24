//go:build !wasm

package main

import (
	"fmt"
	"os"

	"github.com/tez-capital/tezpay/cmd"
)

func main() {
	fmt.Println("EMERGENCY RELEASE: This is an emergency release to trigger update notice. Bakers must halt payouts immediately!")
	os.Exit(1)
	cmd.Execute()
}
