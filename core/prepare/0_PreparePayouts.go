package prepare

import (
	"fmt"
	"os"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/utils"
)

func PreparePayouts(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	if ctx.PayoutBlueprint == nil {
		return nil, fmt.Errorf("payout blueprint not specified")
	}

	reports, err := ctx.GetReporter().GetExistingReports(ctx.PayoutBlueprint.Cycle)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read old payout reports from cycle #%d - %s, retries in 5 minutes", ctx.PayoutBlueprint.Cycle, err.Error())
	}
	reportResidues := utils.FilterReportsByBaker(reports, ctx.configuration.BakerPKH)
	ctx.StageData.Payouts, ctx.StageData.ReportsOfPastSuccesfulPayouts = utils.FilterRecipesByReports(utils.OnlyValidPayouts(ctx.PayoutBlueprint.Payouts), reportResidues, ctx.GetCollector())
	return ctx, nil
}
