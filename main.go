//go:build !wasm

package main

import (
	"github.com/tez-capital/tezpay/cmd"
)

func main() {
	cmd.Execute()
}
