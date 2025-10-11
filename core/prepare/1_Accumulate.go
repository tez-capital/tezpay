package prepare

import (
	"slices"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/extension"
)

type AfterPayoutsAccumulated struct {
	Recipes                       []common.PayoutRecipe             `json:"recipes"`
	AccumulatedPayouts            []*common.AccumulatedPayoutRecipe `json:"payouts"`
	InvalidRecipes                []common.PayoutRecipe             `json:"invalid_payouts"`
	ReportsOfPastSuccesfulPayouts []common.PayoutReport             `json:"reports_of_past_succesful_payouts"`
}

func ExecuteAfterPayoutsAccumulated(data *AfterPayoutsAccumulated) error {
	return extension.ExecuteHook(enums.EXTENSION_HOOK_AFTER_PAYOUTS_ACCUMULATED, "0.1", data)
}

func AccumulatePayouts(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	if ctx.PayoutBlueprints == nil {
		return nil, constants.ErrMissingPayoutBlueprint
	}
	// copy valid payouts to accumulated in case we return early
	ctx.StageData.AccumulatedPayouts = lo.Map(ctx.StageData.Payouts, func(payout common.PayoutRecipe, _ int) *common.AccumulatedPayoutRecipe {
		return payout.AsAccumulated()
	})
	if !options.Accumulate {
		return ctx, nil
	}

	cycles := lo.Reduce(ctx.StageData.Payouts, func(acc []int64, payout common.PayoutRecipe, _ int) []int64 {
		if !slices.Contains(acc, payout.Cycle) {
			acc = append(acc, payout.Cycle)
		}
		return acc
	}, []int64{})
	if len(cycles) <= 1 { // nothing to accumulate
		return ctx, nil
	}

	logger := ctx.logger.With("phase", "accumulate_payouts")
	logger.Info("accumulating payouts")

	payouts := make([]*common.AccumulatedPayoutRecipe, 0, len(ctx.StageData.Payouts))
	grouped := lo.GroupBy(ctx.StageData.Payouts, func(payout common.PayoutRecipe) string {
		return payout.GetIdentifier()
	})

	for k, groupedPayouts := range grouped {
		if k == "" || len(groupedPayouts) <= 1 {
			payouts = append(payouts, lo.Map(groupedPayouts, func(payout common.PayoutRecipe, _ int) *common.AccumulatedPayoutRecipe {
				return payout.AsAccumulated()
			})...)
			continue
		}

		basePayout := groupedPayouts[0].AsAccumulated()
		groupedPayouts = groupedPayouts[1:]
		for _, payout := range groupedPayouts {
			combined, err := basePayout.Add(&payout)
			if err != nil {
				return nil, err
			}
			basePayout = combined
		}

		payouts = append(payouts, basePayout) // add the combined
	}

	hookData := &AfterPayoutsAccumulated{
		Recipes: lo.Reduce(ctx.PayoutBlueprints, func(agg []common.PayoutRecipe, blueprint *common.CyclePayoutBlueprint, _ int) []common.PayoutRecipe {
			return append(agg, blueprint.Payouts...)
		}, make([]common.PayoutRecipe, 0)),
		AccumulatedPayouts:            payouts,
		InvalidRecipes:                ctx.StageData.InvalidRecipes,
		ReportsOfPastSuccesfulPayouts: ctx.StageData.ReportsOfPastSuccesfulPayouts,
	}
	err := ExecuteAfterPayoutsAccumulated(hookData)
	if err != nil {
		return ctx, err
	}

	ctx.StageData.AccumulatedPayouts = hookData.AccumulatedPayouts
	ctx.StageData.InvalidRecipes = hookData.InvalidRecipes
	ctx.StageData.ReportsOfPastSuccesfulPayouts = hookData.ReportsOfPastSuccesfulPayouts
	return ctx, nil
}
