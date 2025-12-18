package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezpay/constants"
	"github.com/trilitech/tzgo/codec"
	"github.com/trilitech/tzgo/rpc"
)

var revealCmd = &cobra.Command{
	Use:   "reveal",
	Short: "reveals the payout wallet",
	Long:  "reveals the payout wallet on the blockchain",
	Run: func(cmd *cobra.Command, args []string) {
		_, _, signer, transactor := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()

		op := codec.NewOp().WithSource(signer.GetPKH())
		op.WithTTL(constants.MAX_OPERATION_TTL)
		reveal := &codec.Reveal{
			Manager: codec.Manager{
				Source: signer.GetPKH(),
			},
			PublicKey: signer.GetKey(),
		}
		reveal.WithLimits(rpc.DefaultRevealLimits)
		op.WithContents(reveal)

		slog.Info("revealing payout wallet", "confirmations_required", constants.DEFAULT_REQUIRED_CONFIRMATIONS)
		opts := rpc.DefaultOptions
		opts.Confirmations = constants.DEFAULT_REQUIRED_CONFIRMATIONS
		opts.Signer = signer.GetSigner()
		opts.Sender = signer.GetPKH()

		rcpt, err := transactor.Send(op, &opts)
		if err != nil {
			slog.Error("failed to confirm tx", "error", err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
		}
		if !rcpt.IsSuccess() {
			slog.Error("tx failed", "error", rcpt.Error().Error())
			fmt.Println(rcpt)
			os.Exit(EXIT_OPERTION_FAILED)
		}
		slog.Info("reveal successful")
	},
}

func init() {
	RootCmd.AddCommand(revealCmd)
}
