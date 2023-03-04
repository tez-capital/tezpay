package generate

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func ValidateSimulatedPayouts(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (result *PayoutGenerationContext, err error) {
	configuration := ctx.GetConfiguration()
	simulated := ctx.StageData.PayoutCandidatesSimulated

	log.Debug("validating simulated payout candidates")

	ctx.StageData.PayoutCandidatesSimulated = lo.Map(simulated, func(candidate PayoutCandidateSimulated, _ int) PayoutCandidateSimulated {
		if candidate.IsInvalid {
			return candidate
		}

		validationContext := candidate.ToValidationContext(configuration)
		result := *validationContext.Validate(
			MinumumAmountSimulatedValidator,
		).ToPayoutCandidateSimulated()

		// collect fees if invalid
		if candidate.IsInvalid {
			ctx.StageData.BakerFeesAmount = ctx.StageData.BakerFeesAmount.Add(candidate.BondsAmount)
			candidate.Fee = candidate.Fee.Add(candidate.BondsAmount)
			candidate.BondsAmount = tezos.Zero
		}
		return result
	})

	return ctx, nil
}
