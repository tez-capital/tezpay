package generate

import (
	"errors"
	"fmt"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/utils"
)

func estimateBatchSerializationGasLimit(ctx *PayoutGenerationContext) error {
	op, err := buildOpForEstimation(ctx, []common.TransferArgs{}, true)
	if err != nil {
		return err
	}
	receipt, err := ctx.GetCollector().Simulate(op, ctx.PayoutKey)
	if err != nil || (receipt != nil && !receipt.IsSuccess()) {
		if receipt != nil && receipt.Error() != nil && (err == nil || receipt.Error().Error() != err.Error()) {
			return errors.Join(receipt.Error(), err)
		}
		return err
	}

	costs := receipt.Op.Costs()
	if len(costs) < 2 {
		utils.PanicWithMetadata("partial estimate", "171037723382b8e880b029bbd881016eb6362a96a13e91e8f25ea9223d02fa31", costs)
	}

	ctx.StageData.BatchMetadataDeserializationGasLimit = costs[0].GasUsed - costs[len(costs)-1].GasUsed

	if ctx.StageData.BatchMetadataDeserializationGasLimit < 0 {
		utils.PanicWithMetadata("unexpected deserialization limit", "171037723382b8e880b029bbd881016eb6362a96a13e91e8f25ea9223d02fa32", ctx.StageData.BatchMetadataDeserializationGasLimit)
	}
	return nil
}

func CheckConditionsAndPrepare(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	collector := ctx.GetCollector()

	payoutAddress := ctx.PayoutKey.Address()
	revealed, err := collector.IsRevealed(payoutAddress)
	if err != nil {
		return ctx, errors.Join(constants.ErrRevealCheckFailed, fmt.Errorf("address - %s", payoutAddress), err)
	}
	if !revealed {
		return ctx, errors.Join(constants.ErrNotRevealed, fmt.Errorf("address - %s", payoutAddress))
	}

	err = estimateBatchSerializationGasLimit(ctx)
	if err != nil {
		return ctx, errors.Join(constants.ErrFailedToEstimateSerializationGasLimit, err)
	}

	return ctx, nil
}
