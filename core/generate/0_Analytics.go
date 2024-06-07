package generate

import (
	"os"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
)

func SendAnalytics(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	configuration := ctx.GetConfiguration()

	if os.Getenv("DISABLE_TEZPAY_ANALYTICS") == "true" {
		return ctx, nil
	}

	if configuration.DisableAnalytics {
		return ctx, nil
	}

	ctx.GetCollector().SendAnalytics(configuration.BakerPKH.String(), constants.VERSION)

	return ctx, nil
}
