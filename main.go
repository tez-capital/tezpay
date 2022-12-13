//go:build !wasm

package main

import (
	"github.com/alis-is/tezpay/cmd"
)

func main() {
	cmd.Execute()
}
