package stages

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/payout/common"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func validateSimulatedPayouts(ctx common.Context) (result common.Context, err error) {
	configuration := ctx.GetConfiguration()
	simulated := ctx.StageData.PayoutCandidatesSimulated

	log.Debug("validating simulated payout candidates")

	// TODO: Accounting
	ctx.StageData.PayoutCandidatesSimulated = lo.Map(simulated, func(candidate common.PayoutCandidateSimulated, _ int) common.PayoutCandidateSimulated {
		if candidate.Candidate.IsInvalid {
			return candidate
		}

		validationContext := candidate.ToValidationContext(configuration)
		result := *validationContext.Validate(
			common.MinumumAmountSimulatedValidator,
		).ToPayoutCandidateSimulated()

		// collect fees if invalid
		if candidate.Candidate.IsInvalid {
			ctx.StageData.BakerFeesAmount = ctx.StageData.BakerFeesAmount.Add(candidate.BondsAmount)
			candidate.Fee = candidate.Fee.Add(candidate.BondsAmount)
			candidate.BondsAmount = tezos.Zero
		}
		return result
	})

	return ctx, nil
}

var ValidateSimulatedPayouts = common.WrapStage(validateSimulatedPayouts)
