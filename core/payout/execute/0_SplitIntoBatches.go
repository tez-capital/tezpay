package execute

import (
	"fmt"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/utils"
)

func SplitIntoBatches(ctx *PayoutExecutionContext, options *common.ExecutePayoutsOptions) (*PayoutExecutionContext, error) {
	var err error
	ctx.StageData.Limits, err = ctx.GetTransactor().GetLimits()
	if err != nil {
		return nil, fmt.Errorf("failed to get tezos chain limits - %s, retries in 5 minutes", err.Error())
	}

	if options.MixInContractCalls {
		ctx.StageData.Batches = common.SplitIntoBatches(ctx.Payouts, ctx.StageData.Limits)
	} else {
		contractBatches := common.SplitIntoBatches(utils.FilterPayoutsByType(ctx.Payouts, tezos.AddressTypeContract), ctx.StageData.Limits)
		txBatches := common.SplitIntoBatches(utils.RejectPayoutsByType(ctx.Payouts, tezos.AddressTypeContract), ctx.StageData.Limits)
		ctx.StageData.Batches = append(txBatches, contractBatches...)
	}
	return ctx, nil
}
