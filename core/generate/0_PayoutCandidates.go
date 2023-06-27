package generate

import (
	"fmt"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/extension"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

type AfterCandidateGeneratedHookData struct {
	Cycle      int64             `json:"cycle"`
	Candidates []PayoutCandidate `json:"candidates"`
}

func ExecuteAfterCandidateGenerated(data *AfterCandidateGeneratedHookData) error {
	return extension.ExecuteHook(enums.EXTENSION_HOOK_AFTER_CANDIDATES_GENERATED, "0.2", data)
}

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

	payoutCandidates := lo.Map(ctx.StageData.CycleData.Delegators, func(delegator common.Delegator, _ int) PayoutCandidate {
		payoutCandidate := DelegatorToPayoutCandidate(delegator, configuration)
		validationContext := payoutCandidate.ToValidationContext(ctx)
		return *validationContext.Validate(
			IsIgnoredValidator,
			RecipientValidator,
			MinimumBalanceValidator,
			IgnoreKtValidator,
			Emptiedalidator,
			RecipientNotBaker,
			NotExcludedByAddressPrefix,
		).ToPayoutCandidate()
	})

	hookData := &AfterCandidateGeneratedHookData{
		Cycle:      options.Cycle,
		Candidates: payoutCandidates,
	}
	err = ExecuteAfterCandidateGenerated(hookData)
	if err != nil {
		return ctx, err
	}
	payoutCandidates = hookData.Candidates

	ctx.StageData.PayoutCandidates = payoutCandidates
	return ctx, nil
}
