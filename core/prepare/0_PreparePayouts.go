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
)

type AfterPayoutsPreapered struct {
	ValidPayouts                  []common.PayoutRecipe `json:"payouts"`
	InvalidPayouts                []common.PayoutRecipe `json:"invalid_payouts"`
	ReportsOfPastSuccesfulPayouts []common.PayoutReport `json:"reports_of_past_succesful_payouts"`
}

func ExecuteAfterPayoutsPrepared(data *AfterPayoutsPreapered) error {
	return extension.ExecuteHook(enums.EXTENSION_HOOK_AFTER_PAYOUTS_PREPARED, "0.1", data)
}

func PreparePayouts(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	var err error
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
		ValidPayouts:                  utils.OnlyValidPayouts(payouts),
		InvalidPayouts:                utils.OnlyInvalidPayouts(payouts),
		ReportsOfPastSuccesfulPayouts: reportsOfPastSuccesfulPayouts,
	}
	err = ExecuteAfterPayoutsPrepared(hookData)
	if err != nil {
		return ctx, err
	}
	ctx.StageData.ValidPayouts, ctx.StageData.InvalidPayouts, ctx.StageData.ReportsOfPastSuccesfulPayouts = hookData.ValidPayouts, hookData.InvalidPayouts, hookData.ReportsOfPastSuccesfulPayouts

	return ctx, nil
}
