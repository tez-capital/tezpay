package generate

import (
	"errors"
	"fmt"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
)

func CheckConditions(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	collector := ctx.GetCollector()

	payoutAddress := ctx.PayoutKey.Address()
	revealed, err := collector.IsRevealed(payoutAddress)
	if err != nil {
		return ctx, errors.Join(constants.ErrRevealCheckFailed, fmt.Errorf("address - %s", payoutAddress), err)
	}
	if !revealed {
		return ctx, errors.Join(constants.ErrNotRevealed, fmt.Errorf("address - %s", payoutAddress))
	}

	return ctx, nil
}
