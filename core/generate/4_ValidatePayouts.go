package generate

import (
	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/utils"
)

func ValidatePayouts(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (result *PayoutGenerationContext, err error) {
	logger := ctx.logger.With("phase", "validate_simulated_payouts")
	logger.Info("validating payout candidates")

	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = lo.Map(ctx.StageData.PayoutCandidatesWithBondAmountAndFees, func(candidate PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateWithBondAmountAndFee {
		validationContext := candidate.ToValidationContext(ctx)
		result := *validationContext.Validate(
			TxKindValidator,
			MinumumAmountValidator,
		).ToPresimPayoutCandidate()

		utils.AssertZAmountPositiveOrZero(candidate.BondsAmount)
		if candidate.IsInvalid {
			ctx.StageData.BakerFeesAmount = ctx.StageData.BakerFeesAmount.Add(candidate.BondsAmount)
			candidate.Fee = candidate.Fee.Add(candidate.BondsAmount) // we need to add because we already collected fees from bonds in 2_CollectBakerFee.go
		}
		return result
	})

	return ctx, nil
}
