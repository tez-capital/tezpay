package stages

import (
	"github.com/alis-is/tezpay/core/common"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func generatePayoutCandidates(ctx Context) (Context, error) {
	configuration := ctx.GetConfiguration()

	log.Debugf("genrating payout candidates")

	ctx.StageData.PayoutCandidates = lo.Map(ctx.CycleData.Delegators, func(delegator common.Delegator, _ int) PayoutCandidate {
		payoutCandidate := DelegatorToPayoutCandidate(delegator, configuration)
		validationContext := payoutCandidate.ToValidationContext(&ctx)
		return *validationContext.Validate(
			IsIgnoredValidator,
			RecipientValidator,
			MinimumBalanceValidator,
			IgnoreKtValidator,
			Emptiedalidator,
			RecipientNotBaker,
		).ToPayoutCandidate()
	})

	return ctx, nil
}

var GeneratePayoutCandidates = WrapStage(generatePayoutCandidates)
