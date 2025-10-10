package generate

import (
	"fmt"
	"log/slog"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/utils"
)

func getDistributionPayouts(logger *slog.Logger, kind enums.EPayoutKind, distributionDefinition map[string]float64, amount tezos.Z, ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) ([]common.PayoutRecipe, error) {
	totalPercentage := lo.Reduce(lo.Values(distributionDefinition), func(agg float64, entry float64, _ int) float64 {
		return agg + entry
	}, float64(0))

	if totalPercentage > 100 {
		return []common.PayoutRecipe{}, fmt.Errorf("expects <= 100%% but only has %f", totalPercentage)
	}

	payouts := make([]common.PayoutRecipe, 0, len(distributionDefinition))
	for recipient, portion := range distributionDefinition {
		recipe := common.PayoutRecipe{
			Baker:   ctx.GetConfiguration().BakerPKH,
			Cycle:   options.Cycle,
			Kind:    kind,
			TxKind:  enums.PAYOUT_TX_KIND_TEZ,
			IsValid: true,
		}
		recipient, err := tezos.ParseAddress(recipient)
		if err != nil {
			recipe.IsValid = false
			recipe.Note = string(enums.INVALID_INVALID_ADDRESS)
			payouts = append(payouts, recipe)
			continue
		}
		recipe.Recipient = recipient
		if recipient.Equal(ctx.PayoutKey.Address()) {
			recipe.IsValid = false
			recipe.Note = string(enums.INVALID_RECIPIENT_TARGETS_PAYOUT)
			payouts = append(payouts, recipe)
			continue
		}

		recipientPortion := utils.GetZPortion(amount, portion)
		recipe.Amount = recipientPortion
		if recipientPortion.IsZero() || recipientPortion.IsNeg() {
			recipe.IsValid = false
			recipe.Note = string(enums.INVALID_PAYOUT_ZERO)
			payouts = append(payouts, recipe)
			continue
		}

		payouts = append(payouts, recipe)
	}

	return payouts, nil
}

// injects bonds, fee and donation payments and finalizes Payouts
func FinalizePayouts(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (result *PayoutGenerationContext, err error) {
	configuration := ctx.GetConfiguration()
	logger := ctx.logger.With("phase", "finalize_payouts")
	logger.Info("finalizing payouts")
	simulated := ctx.StageData.PayoutCandidatesWithBondAmountAndFees

	delegatorPayouts := lo.Map(simulated, func(candidate PayoutCandidateWithBondAmountAndFee, _ int) common.PayoutRecipe {
		return candidate.ToPayoutRecipe(ctx.GetConfiguration().BakerPKH, options.Cycle, enums.PAYOUT_KIND_DELEGATOR_REWARD)
	})

	// bonds
	bondsPayouts, err := getDistributionPayouts(logger, enums.PAYOUT_KIND_BAKER_REWARD, configuration.IncomeRecipients.Bonds, ctx.StageData.BakerBondsAmount, ctx, options)
	if err != nil {
		return ctx, fmt.Errorf("invalid bonds distribution - %s", err.Error())
	}

	// fees
	feesPayouts, err := getDistributionPayouts(logger, enums.PAYOUT_KIND_FEE_INCOME, configuration.IncomeRecipients.Fees, ctx.StageData.BakerFeesAmount, ctx, options)
	if err != nil {
		return ctx, fmt.Errorf("invalid fees distribution - %s", err.Error())
	}

	// donations
	donationDistributionDefinition := configuration.IncomeRecipients.Donations
	if len(donationDistributionDefinition) == 0 && configuration.IncomeRecipients.DonateBonds+configuration.IncomeRecipients.DonateFees > 0 { // inject default destination
		logger.Debug("no donation destination found, donating to tez.capital")
		donationDistributionDefinition = map[string]float64{
			constants.DEFAULT_DONATION_ADDRESS: 100,
		}
	}
	donationPayouts, err := getDistributionPayouts(logger, enums.PAYOUT_KIND_DONATION, donationDistributionDefinition, ctx.StageData.DonateBondsAmount.Add(ctx.StageData.DonateFeesAmount), ctx, options)
	if err != nil {
		return ctx, fmt.Errorf("invalid donation distribution - %s", err.Error())
	}

	payouts := make([]common.PayoutRecipe, 0)
	payouts = append(payouts, delegatorPayouts...)
	payouts = append(payouts, bondsPayouts...)
	payouts = append(payouts, feesPayouts...)
	payouts = append(payouts, donationPayouts...)

	ctx.StageData.Payouts = payouts

	return ctx, nil
}
