package prepare

import (
	"errors"
	"fmt"
	"os"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/extension"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/codec"
	"github.com/trilitech/tzgo/tezos"
)

func buildOpForEstimation[T common.TransferArgs](ctx *PayoutPrepareContext, batch []T, injectBurnTransactions bool) (*codec.Op, error) {
	var err error
	op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
	op.WithTTL(constants.MAX_OPERATION_TTL)
	if injectBurnTransactions {
		op.WithTransfer(tezos.BurnAddress, 1)
	}
	for _, p := range batch {
		if err = common.InjectTransferContents(op, ctx.PayoutKey.Address(), p); err != nil {
			break
		}
	}
	if injectBurnTransactions {
		op.WithTransfer(tezos.BurnAddress, 1)
	}
	return op, err
}

func estimateBatchSerializationGasLimit(ctx *PayoutPrepareContext) error {
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

type AfterPayoutsPreapered struct {
	Recipes                       []common.PayoutRecipe `json:"recipes"`
	Payouts                       []common.PayoutRecipe `json:"payouts"`
	InvalidRecipes                []common.PayoutRecipe `json:"invalid_payouts"`
	ReportsOfPastSuccesfulPayouts []common.PayoutReport `json:"reports_of_past_succesful_payouts"`
}

func ExecuteAfterPayoutsPrepared(data *AfterPayoutsPreapered) error {
	return extension.ExecuteHook(enums.EXTENSION_HOOK_AFTER_PAYOUTS_PREPARED, "0.1", data)
}

func PreparePayouts(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	logger := ctx.logger.With("phase", "prepare_payouts")
	logger.Debug("estimating serialization gas limit")
	var err error

	if err = estimateBatchSerializationGasLimit(ctx); err != nil {
		return ctx, errors.Join(constants.ErrFailedToEstimateSerializationGasLimit, err)
	}

	logger.Info("preparing payouts")
	if ctx.PayoutBlueprints == nil {
		return nil, constants.ErrMissingPayoutBlueprint
	}

	count := lo.Reduce(ctx.PayoutBlueprints, func(agg int, blueprint *common.CyclePayoutBlueprint, _ int) int {
		return agg + len(blueprint.Payouts)
	}, 0)

	payouts := make([]common.PayoutRecipe, 0, count)
	reportsOfPastSuccesfulPayouts := make([]common.PayoutReport, 0, count)
	for _, blueprint := range ctx.PayoutBlueprints {
		reports, err := ctx.GetReporter().GetExistingReports(blueprint.Cycle)
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.Join(constants.ErrPayoutsFromFileLoadFailed, fmt.Errorf("cycle: %d", blueprint.Cycle), err)
		}
		reportResidues := utils.FilterReportsByBaker(reports, ctx.configuration.BakerPKH)
		// we match already paid even against invalid set of payouts in case they were paid under different conditions
		bluePrintPayouts, blueprintReportsOfPastSuccesfulPayouts := utils.FilterRecipesByReports(blueprint.Payouts, reportResidues, ctx.GetCollector())

		payouts = append(payouts, bluePrintPayouts...)
		reportsOfPastSuccesfulPayouts = append(reportsOfPastSuccesfulPayouts, blueprintReportsOfPastSuccesfulPayouts...)
	}

	hookData := &AfterPayoutsPreapered{
		Recipes: lo.Reduce(ctx.PayoutBlueprints, func(agg []common.PayoutRecipe, blueprint *common.CyclePayoutBlueprint, _ int) []common.PayoutRecipe {
			return append(agg, blueprint.Payouts...)
		}, make([]common.PayoutRecipe, 0)),
		Payouts:                       utils.OnlyValidPayoutRecipes(payouts),
		InvalidRecipes:                utils.OnlyInvalidPayoutRecipes(payouts),
		ReportsOfPastSuccesfulPayouts: reportsOfPastSuccesfulPayouts,
	}
	err = ExecuteAfterPayoutsPrepared(hookData)
	if err != nil {
		return ctx, err
	}
	ctx.StageData.Payouts = hookData.Payouts
	ctx.StageData.InvalidRecipes = hookData.InvalidRecipes
	ctx.StageData.ReportsOfPastSuccesfulPayouts = hookData.ReportsOfPastSuccesfulPayouts

	return ctx, nil
}
