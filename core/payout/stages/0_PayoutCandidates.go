package stages

import (
	tezpay_tezos "github.com/alis-is/tezpay/clients/tezos"
	"github.com/alis-is/tezpay/core/payout/common"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func generatePayoutCandidates(ctx common.Context) (common.Context, error) {
	configuration := ctx.GetConfiguration()

	log.Debugf("genrating payout candidates")

	ctx.StageData.PayoutCandidates = lo.Map(ctx.CycleData.Delegators, func(delegator tezpay_tezos.Delegator, _ int) common.PayoutCandidate {
		payoutCandidate := common.DelegatorToPayoutCandidate(delegator, configuration)
		validationContext := payoutCandidate.ToValidationContext(&ctx)
		return *validationContext.Validate(
			common.IsIgnoredValidator,
			common.RecipientValidator,
			common.MinimumBalanceValidator,
			common.IgnoreKtValidator,
			common.Emptiedalidator,
			common.RecipientNotBaker,
		).ToPayoutCandidate()
	})

	return ctx, nil
}

var GeneratePayoutCandidates = common.WrapStage(generatePayoutCandidates)
