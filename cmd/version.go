//go:build !wasm

package cmd

import (
	"fmt"

	"github.com/alis-is/tezpay/constants"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "prints tezpay version",
	Long:  "generates payouts without further processing",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(constants.VERSION)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
