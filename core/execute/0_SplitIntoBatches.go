package execute

import (
	"fmt"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/utils"
)

func SplitIntoBatches(ctx *PayoutExecutionContext, options *common.ExecutePayoutsOptions) (*PayoutExecutionContext, error) {
	var err error
	ctx.StageData.Limits, err = ctx.GetTransactor().GetLimits()
	if err != nil {
		return nil, fmt.Errorf("failed to get tezos chain limits - %s, retries in 5 minutes", err.Error())
	}
	payouts := utils.OnlyValidPayouts(ctx.Payouts) // make sure we are batching only valid payouts
	payoutsWithoutFa := utils.RejectPayoutsByTxKind(payouts, enums.FA_OPERATION_KINDS)

	faRecipes := utils.FilterPayoutsByTxKind(payouts, enums.FA_OPERATION_KINDS)
	contractTezRecipes := utils.FilterPayoutsByType(payoutsWithoutFa, tezos.AddressTypeContract)
	classicTezRecipes := utils.RejectPayoutsByType(payoutsWithoutFa, tezos.AddressTypeContract)

	toBatch := make([][]common.PayoutRecipe, 0, 3)
	if options.MixInFATransfers {
		classicTezRecipes = append(classicTezRecipes, faRecipes...)
	} else {
		toBatch = append(toBatch, faRecipes)
	}
	if options.MixInContractCalls {
		classicTezRecipes = append(classicTezRecipes, contractTezRecipes...)
	} else {
		toBatch = append(toBatch, contractTezRecipes)
	}
	toBatch = append(toBatch, classicTezRecipes)

	stageBatches := make([]common.RecipeBatch, 0)
	for _, batch := range toBatch {
		batches, err := common.SplitIntoBatches(batch, ctx.StageData.Limits)
		if err != nil {
			return nil, err
		}
		stageBatches = append(stageBatches, batches...)
	}

	ctx.StageData.Batches = stageBatches
	return ctx, nil
}
