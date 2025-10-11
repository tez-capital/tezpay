package prepare

import (
	"fmt"
	"log/slog"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/core/estimate"
	"github.com/trilitech/tzgo/tezos"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/utils"
)

func getDistributionPayouts(logger *slog.Logger, kind enums.EPayoutKind, distributionDefinition map[string]float64, amount tezos.Z, ctx *PayoutPrepareContext, cycle int64) ([]common.PayoutRecipe, error) {
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
			Cycle:   cycle,
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

	estimateContext := &estimate.EstimationContext{
		PayoutKey:                            ctx.PayoutKey,
		Collector:                            ctx.GetCollector(),
		Configuration:                        ctx.GetConfiguration(),
		BatchMetadataDeserializationGasLimit: ctx.StageData.BatchMetadataDeserializationGasLimit,
	}

	all := lo.Map(estimate.EstimateTransactionFees(utils.MapToPointers(payouts), estimateContext), func(result estimate.EstimateResult[*common.PayoutRecipe], _ int) common.PayoutRecipe {
		if result.Error != nil {
			logger.Warn("failed to estimate tx costs", "recipient", result.Transaction.Recipient, "delegator", ctx.PayoutKey.Address(), "amount", result.Transaction.Amount.Int64(), "kind", result.Transaction.TxKind, "error", result.Error)
			result.Transaction.IsValid = false
			result.Transaction.Note = string(enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS)
		}
		result.Transaction.OpLimits = result.OpLimits
		return *result.Transaction
	})
	return all, nil
}

// injects bonds, fee and donation payments and finalizes Payouts
func FinalizePayouts(ctx *PayoutPrepareContext, options *common.PreparePayoutsOptions) (result *PayoutPrepareContext, err error) {
	configuration := ctx.GetConfiguration()
	logger := ctx.logger.With("phase", "finalize_payouts")
	logger.Info("finalizing payouts")

	lastCycle := lo.Reduce(ctx.StageData.AccumulatedPayouts, func(acc int64, payout *common.AccumulatedPayoutRecipe, _ int) int64 {
		return max(acc, payout.Cycle)
	}, 0)

	feesFromNewTransfers := lo.Reduce(ctx.StageData.AccumulatedPayouts, func(acc tezos.Z, payout *common.AccumulatedPayoutRecipe, _ int) tezos.Z {
		if payout.TxKind != enums.PAYOUT_TX_KIND_TEZ || payout.Kind != enums.PAYOUT_KIND_DELEGATOR_REWARD {
			return acc
		}
		return acc.Add(payout.Fee)
	}, tezos.Zero)
	donateFeesAmount := utils.GetZPortion(feesFromNewTransfers, configuration.IncomeRecipients.DonateFees)
	keptFees := feesFromNewTransfers.Sub(donateFeesAmount)

	// fees
	feesPayouts, err := getDistributionPayouts(logger, enums.PAYOUT_KIND_FEE_INCOME, configuration.IncomeRecipients.Fees, keptFees, ctx, lastCycle)
	if err != nil {
		return ctx, fmt.Errorf("invalid fees distribution - %s", err.Error())
	}

	// donations
	donationDistributionDefinition := configuration.IncomeRecipients.Donations
	if len(donationDistributionDefinition) == 0 && configuration.IncomeRecipients.DonateFees > 0 { // inject default destination
		logger.Debug("no donation destination found, donating to tez.capital")
		donationDistributionDefinition = map[string]float64{
			constants.DEFAULT_DONATION_ADDRESS: 100,
		}
	}
	donationPayouts, err := getDistributionPayouts(logger, enums.PAYOUT_KIND_DONATION, donationDistributionDefinition, donateFeesAmount, ctx, lastCycle)
	if err != nil {
		return ctx, fmt.Errorf("invalid donation distribution - %s", err.Error())
	}
	newRecipes := make([]*common.AccumulatedPayoutRecipe, 0, len(feesPayouts)+len(donationPayouts))
	// TODO: restimate accumulated to reduce fees?
	for _, recipe := range feesPayouts {
		existingDestination, found := lo.Find(ctx.StageData.AccumulatedPayouts, func(r *common.AccumulatedPayoutRecipe) bool {
			return r.GetIdentifier() == recipe.GetIdentifier()
		})
		if found {
			existingDestination.Add(&recipe)
		} else {
			newRecipes = append(newRecipes, recipe.AsAccumulated())
		}
	}
	for _, recipe := range donationPayouts {
		existingDestination, found := lo.Find(ctx.StageData.AccumulatedPayouts, func(r *common.AccumulatedPayoutRecipe) bool {
			fmt.Println(r.Kind, recipe.Kind, r.GetIdentifier(), recipe.GetIdentifier())
			return r.GetIdentifier() == recipe.GetIdentifier()
		})
		if found {
			existingDestination.Add(&recipe)
		} else {
			newRecipes = append(newRecipes, recipe.AsAccumulated())
		}
	}
	ctx.StageData.AccumulatedPayouts = append(ctx.StageData.AccumulatedPayouts, newRecipes...)

	return ctx, nil
}
