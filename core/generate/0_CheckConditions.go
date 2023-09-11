package generate

import (
	"errors"
	"fmt"

	"github.com/alis-is/tezpay/common"
)

func CheckConditions(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	collector := ctx.GetCollector()

	payoutAddress := ctx.PayoutKey.Address()
	revealed, err := collector.IsRevealed(payoutAddress)
	if err != nil {
		return ctx, errors.Join(fmt.Errorf("failed to check if payout address - %s - is revealed", payoutAddress), err)
	}
	if !revealed {
		return ctx, fmt.Errorf("payout address - %s - is not revealed", payoutAddress)
	}

	return ctx, nil
}
