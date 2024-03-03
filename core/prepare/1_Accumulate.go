package prepare

import (
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/samber/lo"
)

func AccumulatePayouts(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (*PayoutPrepareContext, error) {
	if ctx.PayoutBlueprints == nil {
		return nil, constants.ErrMissingPayoutBlueprint
	}
	if !options.Accumulate {
		return ctx, nil
	}

	payouts := make([]common.PayoutRecipe, 0, len(ctx.StageData.ValidPayouts))
	accumulatedPayouts := make([]common.PayoutRecipe, 0, len(ctx.StageData.ValidPayouts))
	grouped := lo.GroupBy(ctx.StageData.ValidPayouts, func(payout common.PayoutRecipe) string {
		return payout.GetIdentifier()
	})

	for k, payouts := range grouped {
		if k == "" || len(payouts) <= 1 {
			accumulatedPayouts = append(accumulatedPayouts, payouts...)
			continue
		}

		basePayout := payouts[0]
		payouts = payouts[1:]
		for _, payout := range payouts {
			combined, err := basePayout.Combine(&payout)
			if err != nil {
				return nil, err
			}
			basePayout = *combined
		}
		// TODO: optimize fees and adjust blueprints accordingly

		payouts = append(payouts, basePayout)                       // add the combined
		accumulatedPayouts = append(accumulatedPayouts, payouts...) // add the rest - marked as ACCUMULATED
	}

	ctx.StageData.ValidPayouts = payouts
	ctx.StageData.AccumulatedPayouts = accumulatedPayouts

	return ctx, nil
}
