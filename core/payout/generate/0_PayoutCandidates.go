package generate

import (
	"fmt"

	"github.com/alis-is/tezpay/core/common"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

// func AdjustThroughExtensions(ctx Context, extensions []common.Extension) Context {
// 	for _, extension := range extensions {
// 		ctx = extension.AdjustPayoutCandidates(ctx)
// 	}
// 	return ctx
// }

func GeneratePayoutCandidates(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	configuration := ctx.GetConfiguration()

	if options.Cycle == 0 {
		cycle, err := ctx.GetCollector().GetLastCompletedCycle()
		if err != nil {
			return ctx, err
		}
		options.Cycle = cycle
	}
	log.Infof("generating payouts for cycle %d (baker: '%s')", options.Cycle, configuration.BakerPKH)

	log.Infof("collecting rewards split through %s collector", ctx.GetCollector().GetId())
	var err error
	ctx.StageData.CycleData, err = ctx.GetCollector().GetCycleData(configuration.BakerPKH, options.Cycle)
	if err != nil {
		return ctx, fmt.Errorf("failed to collect cycle data through collector %s - %s", ctx.GetCollector().GetId(), err.Error())
	}

	log.Debugf("genrating payout candidates")

	ctx.StageData.PayoutCandidates = lo.Map(ctx.StageData.CycleData.Delegators, func(delegator common.Delegator, _ int) PayoutCandidate {
		payoutCandidate := DelegatorToPayoutCandidate(delegator, configuration)
		validationContext := payoutCandidate.ToValidationContext(ctx)
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
