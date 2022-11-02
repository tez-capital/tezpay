package cmd

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var transferCmd = &cobra.Command{
	Use:   "transfer <destination> <amount tez>",
	Short: "transfers tez to specified address",
	Long:  "transfers tez to specified address from payout wallet",
	Run: func(cmd *cobra.Command, args []string) {
		_, _, signer, transactor := assertRunWithResult(loadConfigurationAndEngines, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		mutez, _ := cmd.Flags().GetBool(MUTEZ_FLAG)

		if len(args)%2 != 0 {
			log.Error("invalid number of arguments (expects pairs of destination and amount)")
			os.Exit(EXIT_IVNALID_ARGS)
		}
		total := int64(0)

		destinations := make([]string, 0)

		op := codec.NewOp().WithSource(signer.GetPKH())
		for i := 0; i < len(args); i += 2 {
			destination, err := tezos.ParseAddress(args[i])
			if err != nil {
				log.Errorf("invalid destination address '%s' - '%s'", args[i], err.Error())
				os.Exit(EXIT_IVNALID_ARGS)
			}

			amount, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				log.Errorf("invalid amount '%s' - '%s'", args[i+1], err.Error())
				os.Exit(EXIT_IVNALID_ARGS)
			}
			if !mutez {
				amount *= constants.MUTEZ_FACTOR
			}

			mutez := int64(math.Floor(amount))
			total += mutez
			destinations = append(destinations, destination.String())
			op.WithTransfer(destination, mutez)
		}

		if err := requireConfirmation(fmt.Sprintf("do you really want to transfer %s to %s", utils.MutezToTezS(total), strings.Join(destinations, ", "))); err != nil {
			os.Exit(EXIT_OPERTION_CANCELED)
		}
		log.Infof("transfering tez... waiting for %d confirmations", constants.DEFAULT_REQUIRED_CONFIRMATIONS)
		opts := rpc.DefaultOptions
		opts.Confirmations = constants.DEFAULT_REQUIRED_CONFIRMATIONS
		opts.Signer = signer.GetSigner()

		rcpt, err := transactor.Send(op, &opts)
		if err != nil {
			log.Errorf("failed to confirm tx - %s", err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
		}
		if !rcpt.IsSuccess() {
			log.Errorf("tx failed - %s", rcpt.Error().Error())
			os.Exit(EXIT_OPERTION_FAILED)
		}
		log.Info("transfer successful")
	},
}

func init() {
	transferCmd.Flags().Bool(MUTEZ_FLAG, false, "amount in mutez")
	transferCmd.Flags().Bool(CONFIRM_FLAG, false, "automatically confirms transfer")
	RootCmd.AddCommand(transferCmd)
}
