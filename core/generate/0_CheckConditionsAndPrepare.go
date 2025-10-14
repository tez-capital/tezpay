package generate

import (
	"errors"
	"fmt"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
)

func CheckConditionsAndPrepare(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	collector := ctx.GetCollector()
	logger := ctx.logger.With("phase", "check_conditions_and_prepare")
	logger.Info("checking conditions and preparing")
	logger.Debug("checking if payout address is revealed")
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
