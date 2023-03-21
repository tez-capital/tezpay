package prepare

import (
	"fmt"
	"os"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/extension"
	"github.com/alis-is/tezpay/utils"
)

type AfterPayoutsPreapered struct {
	Payouts                       []common.PayoutRecipe `json:"payouts"`
	ReportsOfPastSuccesfulPayouts []common.PayoutReport `json:"reports_of_past_succesful_payouts"`
}

func ExecuteAfterPayoutsPrepared(data *AfterPayoutsPreapered) error {
	return extension.ExecuteHook(enums.EXTENSION_HOOK_AFTER_PAYOUTS_PREPARED, "0.1", data)
}

func PreparePayouts(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	if ctx.PayoutBlueprint == nil {
		return nil, fmt.Errorf("payout blueprint not specified")
	}

	reports, err := ctx.GetReporter().GetExistingReports(ctx.PayoutBlueprint.Cycle)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read old payout reports from cycle #%d - %s, retries in 5 minutes", ctx.PayoutBlueprint.Cycle, err.Error())
	}
	reportResidues := utils.FilterReportsByBaker(reports, ctx.configuration.BakerPKH)
	// we match already paid even against invalid set of payouts in case they were paid under different conditions
	payouts, reportsOfPastSuccesfulPayouts := utils.FilterRecipesByReports(ctx.PayoutBlueprint.Payouts, reportResidues, ctx.GetCollector())

	hookData := &AfterPayoutsPreapered{
		Payouts:                       payouts,
		ReportsOfPastSuccesfulPayouts: reportsOfPastSuccesfulPayouts,
	}
	err = ExecuteAfterPayoutsPrepared(hookData)
	if err != nil {
		return ctx, err
	}
	ctx.StageData.Payouts, ctx.StageData.ReportsOfPastSuccesfulPayouts = hookData.Payouts, hookData.ReportsOfPastSuccesfulPayouts

	return ctx, nil
}
