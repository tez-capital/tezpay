package stages

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/payout/common"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func collectBakerFees(ctx common.Context) (common.Context, error) {
	configuration := ctx.GetConfiguration()
	log.Debugf("collecting baker fee")
	candidates := ctx.StageData.PayoutCandidatesWithBondAmount

	candidatesWithBondsAndFees := lo.Map(candidates, func(candidateWithBondsAmount common.PayoutCandidateWithBondAmount, _ int) common.PayoutCandidateWithBondAmountAndFee {
		if candidateWithBondsAmount.Candidate.IsInvalid {
			return common.PayoutCandidateWithBondAmountAndFee{
				Candidate: candidateWithBondsAmount.Candidate,
			}
		}
		fee := utils.GetZPortion(candidateWithBondsAmount.BondsAmount, candidateWithBondsAmount.Candidate.FeeRate)
		return common.PayoutCandidateWithBondAmountAndFee{
			Candidate:   candidateWithBondsAmount.Candidate,
			BondsAmount: candidateWithBondsAmount.BondsAmount.Sub(fee),
			Fee:         fee,
		}
	})

	collectedFees := lo.Reduce(candidatesWithBondsAndFees, func(agg tezos.Z, candidateWithBondsAmountAndFee common.PayoutCandidateWithBondAmountAndFee, _ int) tezos.Z {
		return agg.Add(candidateWithBondsAmountAndFee.Fee)
	}, tezos.Zero)

	feesDonate := utils.GetZPortion(collectedFees, configuration.IncomeRecipients.Donate)
	ctx.StageData.BakerFeesAmount = collectedFees.Sub(feesDonate)
	ctx.StageData.DonateFeesAmount = feesDonate
	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = candidatesWithBondsAndFees

	return ctx, nil
}

var CollectBakerFee = common.WrapStage(collectBakerFees)
