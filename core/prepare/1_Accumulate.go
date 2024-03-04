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

	for k, groupedPayouts := range grouped {
		if k == "" || len(groupedPayouts) <= 1 {
			payouts = append(payouts, groupedPayouts...)
			continue
		}

		basePayout := groupedPayouts[0]
		groupedPayouts = groupedPayouts[1:]
		for _, payout := range groupedPayouts {
			combined, err := basePayout.Combine(&payout)
			if err != nil {
				return nil, err
			}
			accumulatedPayouts = append(accumulatedPayouts, payout)
			basePayout = *combined
		}
		// TODO: optimize fees and adjust blueprints accordingly

		payouts = append(payouts, basePayout) // add the combined
		// add the rest - marked as ACCUMULATED
	}

	ctx.StageData.ValidPayouts = payouts
	ctx.StageData.AccumulatedPayouts = accumulatedPayouts

	return ctx, nil
}
