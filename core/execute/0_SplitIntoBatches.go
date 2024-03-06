package execute

import (
	"errors"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
)

func splitIntoBatches(payouts []common.PayoutRecipe, limits *common.OperationLimits, metadataDeserializationGasLimit int64) ([]common.RecipeBatch, error) {
	batches := make([]common.RecipeBatch, 0)
	batchBlueprint := common.NewBatch(limits, metadataDeserializationGasLimit)

	for _, payout := range payouts {
		if !batchBlueprint.AddPayout(payout) {
			batches = append(batches, batchBlueprint.ToBatch())
			batchBlueprint = common.NewBatch(limits, metadataDeserializationGasLimit)
			if !batchBlueprint.AddPayout(payout) {
				return nil, constants.ErrPayoutDidNotFitTheBatch
			}
		}
	}
	// append last
	batches = append(batches, batchBlueprint.ToBatch())

	return lo.Filter(batches, func(batch common.RecipeBatch, _ int) bool {
		return len(batch) > 0
	}), nil
}

func SplitIntoBatches(ctx *PayoutExecutionContext, options *common.ExecutePayoutsOptions) (*PayoutExecutionContext, error) {
	var err error
	ctx.StageData.Limits, err = ctx.GetTransactor().GetLimits()
	if err != nil {
		return nil, errors.Join(constants.ErrGetChainLimitsFailed, err)
	}
	payouts := ctx.ValidPayouts
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

	batchMetadataDeserializationGasLimit := lo.Reduce(ctx.PayoutBlueprints, func(agg int64, blueprint *common.CyclePayoutBlueprint, _ int) int64 {
		return max(agg, blueprint.BatchMetadataDeserializationGasLimit)
	}, 0)

	stageBatches := make([]common.RecipeBatch, 0)
	for _, batch := range toBatch {
		batches, err := splitIntoBatches(batch, ctx.StageData.Limits, batchMetadataDeserializationGasLimit)
		if err != nil {
			return nil, err
		}
		stageBatches = append(stageBatches, batches...)
	}

	ctx.StageData.Batches = stageBatches
	return ctx, nil
}
