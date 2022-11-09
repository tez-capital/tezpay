package stages

import (
	"fmt"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func checkSufficientBalance(ctx Context) (Context, error) {
	configuration := ctx.GetConfiguration()
	log.Debugf("checking for sufficient balance")
	candidates := ctx.StageData.PayoutCandidatesWithBondAmount

	payableBalance, err := ctx.Collector.GetBalance(ctx.PayoutKey.Address())
	if err != nil {
		return ctx, err
	}

	totalPayouts := len(lo.Filter(candidates, func(candidate PayoutCandidateWithBondAmount, _ int) bool {
		return !candidate.IsInvalid
	}))
	// add all bonds, fees and donations destinations
	totalPayouts = totalPayouts + len(configuration.IncomeRecipients.Bonds) + len(configuration.IncomeRecipients.Fees) + utils.Max(len(configuration.IncomeRecipients.Donations), 1)

	requiredbalance := lo.Reduce(candidates, func(agg tezos.Z, candidate PayoutCandidateWithBondAmount, _ int) tezos.Z {
		return agg.Add(candidate.BondsAmount)
	}, tezos.Zero)

	requiredbalance = ctx.StageData.BakerBondsAmount.Add(requiredbalance)
	requiredbalance = requiredbalance.Add(tezos.NewZ(constants.PAYOUT_FEE_BUFFER).Mul64(int64(totalPayouts)))

	diff := payableBalance.Sub(requiredbalance)
	if diff.IsNeg() || diff.IsZero() { // zero is probably too on edge so better to keep checking for zero
		return ctx, fmt.Errorf("insufficient balance - needs at least %s but only has %s", utils.MutezToTezS(requiredbalance.Int64()), utils.MutezToTezS(payableBalance.Int64()))
	}

	return ctx, nil
}

var CheckSufficientBalance = WrapStage(checkSufficientBalance)
