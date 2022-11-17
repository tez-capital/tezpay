package stages

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func collectBakerFees(ctx Context) (Context, error) {
	configuration := ctx.GetConfiguration()
	log.Debugf("collecting baker fee")
	candidates := ctx.StageData.PayoutCandidatesWithBondAmount

	candidatesWithBondsAndFees := lo.Map(candidates, func(candidateWithBondsAmount PayoutCandidateWithBondAmount, _ int) PayoutCandidateWithBondAmountAndFee {
		if candidateWithBondsAmount.IsInvalid {
			return PayoutCandidateWithBondAmountAndFee{
				PayoutCandidateWithBondAmount: candidateWithBondsAmount,
			}
		}
		fee := utils.GetZPortion(candidateWithBondsAmount.BondsAmount, candidateWithBondsAmount.FeeRate)
		candidateWithBondsAmount.BondsAmount = candidateWithBondsAmount.BondsAmount.Sub(fee)
		if candidateWithBondsAmount.BondsAmount.IsZero() || candidateWithBondsAmount.BondsAmount.IsNeg() {
			candidateWithBondsAmount.IsInvalid = true
			candidateWithBondsAmount.InvalidBecause = enums.INVALID_PAYOUT_BELLOW_MINIMUM
			return PayoutCandidateWithBondAmountAndFee{
				PayoutCandidateWithBondAmount: candidateWithBondsAmount,
			}
		}
		return PayoutCandidateWithBondAmountAndFee{
			PayoutCandidateWithBondAmount: candidateWithBondsAmount,
			Fee:                           fee,
		}
	})

	collectedFees := lo.Reduce(candidatesWithBondsAndFees, func(agg tezos.Z, candidateWithBondsAmountAndFee PayoutCandidateWithBondAmountAndFee, _ int) tezos.Z {
		return agg.Add(candidateWithBondsAmountAndFee.Fee)
	}, tezos.Zero)

	feesDonate := utils.GetZPortion(collectedFees, configuration.IncomeRecipients.Donate)
	ctx.StageData.BakerFeesAmount = collectedFees.Sub(feesDonate)
	ctx.StageData.DonateFeesAmount = feesDonate
	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = candidatesWithBondsAndFees

	return ctx, nil
}

var CollectBakerFee = WrapStage(collectBakerFees)
