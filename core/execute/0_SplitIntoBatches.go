package execute

import (
	"errors"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/tezos"
)

func splitIntoBatches(payouts []*common.AccumulatedPayoutRecipe, limits *common.OperationLimits, metadataDeserializationGasLimit int64) ([]common.RecipeBatch, error) {
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
	logger := ctx.logger.With("phase", "split_into_batches")
	logger.Info("splitting into batches")
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

	toBatch := make([][]*common.AccumulatedPayoutRecipe, 0, 3)
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
		batches, err := splitIntoBatches(batch, ctx.StageData.Limits, ctx.BatchMetadataDeserializationGasLimit)
		if err != nil {
			return nil, err
		}
		stageBatches = append(stageBatches, batches...)
	}

	ctx.StageData.Batches = stageBatches
	return ctx, nil
}
