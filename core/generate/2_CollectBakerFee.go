package generate

import (
	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/extension"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/tezos"
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
	logger := ctx.logger.With("phase", "collect_baker_fee")
	logger.Debug("collecting baker fee")
	candidates := ctx.StageData.PayoutCandidatesWithBondAmount

	candidatesWithBondsAndFees := lo.Map(candidates, func(candidateWithBondsAmount PayoutCandidateWithBondAmount, _ int) PayoutCandidateWithBondAmountAndFee {
		if candidateWithBondsAmount.IsInvalid {
			return PayoutCandidateWithBondAmountAndFee{
				PayoutCandidateWithBondAmount: candidateWithBondsAmount,
			}
		}

		if candidateWithBondsAmount.TxKind != enums.PAYOUT_TX_KIND_TEZ {
			logger.Debug("skipping fee collection for non tezos payout", "delegate", candidateWithBondsAmount.Source, "tx_kind", candidateWithBondsAmount.TxKind)
			return PayoutCandidateWithBondAmountAndFee{
				PayoutCandidateWithBondAmount: candidateWithBondsAmount,
			}
		}

		fee := utils.GetZPortion(candidateWithBondsAmount.BondsAmount, candidateWithBondsAmount.FeeRate)
		candidateWithBondsAmount.BondsAmount = candidateWithBondsAmount.BondsAmount.Sub(fee)
		if candidateWithBondsAmount.BondsAmount.IsZero() || candidateWithBondsAmount.BondsAmount.IsNeg() {
			candidateWithBondsAmount.IsInvalid = true
			candidateWithBondsAmount.InvalidBecause = enums.INVALID_NOT_ENOUGH_BONDS_FOR_BAKER_FEE
			candidateWithBondsAmount.BondsAmount = tezos.Zero // this is to prevent negative bonds amount
		}
		utils.AssertZAmountPositiveOrZero(candidateWithBondsAmount.BondsAmount)

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
