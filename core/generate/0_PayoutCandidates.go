package generate

import (
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/extension"
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
	logger := ctx.logger.With("phase", "generate_payout_candidates")
	logger.Info("generating payouts", "cycle", options.Cycle, "baker", configuration.BakerPKH.String())

	if options.Cycle == 0 {
		cycle, err := ctx.GetCollector().GetLastCompletedCycle()
		if err != nil {
			return ctx, err
		}
		options.Cycle = cycle
	}

	logger.Debug("collecting rewards split", "collector", ctx.GetCollector().GetId())
	var err error
	ctx.StageData.CycleData, err = ctx.GetCollector().GetCycleStakingData(configuration.BakerPKH, options.Cycle)
	if err != nil {
		return ctx, errors.Join(constants.ErrCycleDataCollectionFailed, fmt.Errorf("collector: %s", ctx.GetCollector().GetId()), err)
	}

	logger.Debug("generating payout candidates")
	payoutCandidates := lo.Map(ctx.StageData.CycleData.Delegators, func(delegator common.Delegator, _ int) PayoutCandidate {
		payoutCandidate := DelegatorToPayoutCandidate(delegator, configuration)
		validationContext := payoutCandidate.ToValidationContext(ctx)
		return *validationContext.Validate(
			IsIgnoredValidator,
			IsPrefilteredValidator,
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
