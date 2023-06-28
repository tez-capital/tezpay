package generate

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/extension"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

type OnFeesCollectionHookData = struct {
	Cycle      int64                                 `json:"cycle"`
	Candidates []PayoutCandidateWithBondAmountAndFee `json:"candidates"`
}

func ExecuteOnFeesCollection(data *OnFeesCollectionHookData) error {
	return extension.ExecuteHook(enums.EXTENSION_HOOK_ON_FEES_COLLECTION, "0.2", data)
}

func CollectBakerFee(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	configuration := ctx.GetConfiguration()
	log.Debugf("collecting baker fee")
	candidates := ctx.StageData.PayoutCandidatesWithBondAmount

	candidatesWithBondsAndFees := lo.Map(candidates, func(candidateWithBondsAmount PayoutCandidateWithBondAmount, _ int) PayoutCandidateWithBondAmountAndFee {
		if candidateWithBondsAmount.IsInvalid {
			return PayoutCandidateWithBondAmountAndFee{
				PayoutCandidateWithBondAmount: candidateWithBondsAmount,
			}
		}

		if candidateWithBondsAmount.TxKind != enums.PAYOUT_TX_KIND_TEZ {
			log.Trace("skipping fee collection for non tezos payout")
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

	hookData := &OnFeesCollectionHookData{
		Cycle:      options.Cycle,
		Candidates: candidatesWithBondsAndFees,
	}
	err := ExecuteOnFeesCollection(hookData)
	if err != nil {
		return ctx, err
	}
	candidatesWithBondsAndFees = hookData.Candidates

	collectedFees := lo.Reduce(candidatesWithBondsAndFees, func(agg tezos.Z, candidateWithBondsAmountAndFee PayoutCandidateWithBondAmountAndFee, _ int) tezos.Z {
		return agg.Add(candidateWithBondsAmountAndFee.Fee)
	}, tezos.Zero)

	feesDonate := utils.GetZPortion(collectedFees, configuration.IncomeRecipients.DonateFees)
	ctx.StageData.BakerFeesAmount = collectedFees.Sub(feesDonate)
	ctx.StageData.DonateFeesAmount = feesDonate
	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = candidatesWithBondsAndFees

	return ctx, nil
}
