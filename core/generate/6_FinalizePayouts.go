package generate

import (
	"fmt"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"

	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func getDistributionPayouts(kind enums.EPayoutKind, distributionDefinition map[string]float64, amount tezos.Z, ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) ([]common.PayoutRecipe, error) {
	totalPercentage := lo.Reduce(lo.Values(distributionDefinition), func(agg float64, entry float64, _ int) float64 {
		return agg + entry
	}, float64(0))
	if totalPercentage > 100 {
		return []common.PayoutRecipe{}, fmt.Errorf("expects <= 100%% but only has %f", totalPercentage)
	}

	result := make([]common.PayoutRecipe, 0, len(distributionDefinition))
	for recipient, portion := range distributionDefinition {
		recipe := common.PayoutRecipe{
			Baker:   ctx.GetConfiguration().BakerPKH,
			Cycle:   options.Cycle,
			Kind:    kind,
			IsValid: true,
		}
		recipient, err := tezos.ParseAddress(recipient)
		if err != nil {
			recipe.IsValid = false
			recipe.Note = string(enums.INVALID_INVALID_ADDRESS)
			result = append(result, recipe)
			continue
		}
		recipe.Recipient = recipient
		if recipient.Equal(ctx.PayoutKey.Address()) {
			recipe.IsValid = false
			recipe.Note = string(enums.INVALID_RECIPIENT_TARGETS_PAYOUT)
			result = append(result, recipe)
			continue
		}

		recipientPortion := utils.GetZPortion(amount, portion)
		recipe.Amount = recipientPortion
		if recipientPortion.IsZero() || recipientPortion.IsNeg() {
			recipe.IsValid = false
			recipe.Note = string(enums.INVALID_PAYOUT_ZERO)
			result = append(result, recipe)
			continue
		}

		// simulate - because of batch spliting
		op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
		op.WithTTL(constants.MAX_OPERATION_TTL)
		op.WithTransfer(tezos.BurnAddress, 1)
		op.WithTransfer(recipient, recipientPortion.Int64())
		op.WithTransfer(tezos.BurnAddress, 1)

		receipt, err := ctx.GetCollector().Simulate(op, ctx.PayoutKey)

		if err != nil {
			return []common.PayoutRecipe{}, err
		}
		costs := receipt.Costs()
		if len(costs) < 3 {
			return []common.PayoutRecipe{}, fmt.Errorf("invalid costs length, cannot estimate")
		}

		// we use entire serialization cost even with two burn txs, it is used as some offset to avoid exhaustion
		serializationGas := (costs[0].GasUsed - costs[len(costs)-1].GasUsed)
		op.Contents = op.Contents[1 : len(op.Contents)-1]
		costs = costs[1 : len(costs)-1]
		cost := costs[0]

		feeBuffer := ctx.configuration.PayoutConfiguration.TxFeeBuffer
		if recipient.IsContract() {
			feeBuffer = ctx.configuration.PayoutConfiguration.KtTxFeeBuffer
		}

		recipe.OpLimits = &common.OpLimits{
			GasLimit:              cost.GasUsed + ctx.configuration.PayoutConfiguration.TxGasLimitBuffer,
			StorageLimit:          utils.CalculateStorageLimit(cost),
			TransactionFee:        utils.EstimateTransactionFee(op, costs, serializationGas+ctx.configuration.PayoutConfiguration.TxDeserializationGasBuffer+ctx.configuration.PayoutConfiguration.TxGasLimitBuffer) + feeBuffer,
			SerializationGasLimit: serializationGas + ctx.configuration.PayoutConfiguration.TxDeserializationGasBuffer,
		}
		result = append(result, recipe)
	}
	return result, nil
}

// injects bonds, fee and donation payments and finalizes Payouts
func FinalizePayouts(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (result *PayoutGenerationContext, err error) {
	configuration := ctx.GetConfiguration()
	log.Debug("finalizing payouts")
	simulated := ctx.StageData.PayoutCandidatesSimulated

	delegatorPayouts := lo.Map(simulated, func(candidate PayoutCandidateSimulated, _ int) common.PayoutRecipe {
		return candidate.ToPayoutRecipe(ctx.GetConfiguration().BakerPKH, options.Cycle, enums.PAYOUT_KIND_DELEGATOR_REWARD)
	})

	// bonds
	bondsPayouts, err := getDistributionPayouts(enums.PAYOUT_KIND_BAKER_REWARD, configuration.IncomeRecipients.Bonds, ctx.StageData.BakerBondsAmount, ctx, options)
	if err != nil {
		return ctx, fmt.Errorf("invalid bonds distribution - %s", err.Error())
	}

	// fees
	feesPayouts, err := getDistributionPayouts(enums.PAYOUT_KIND_FEE_INCOME, configuration.IncomeRecipients.Fees, ctx.StageData.BakerFeesAmount, ctx, options)
	if err != nil {
		return ctx, fmt.Errorf("invalid fees distribution - %s", err.Error())
	}

	// donations
	donationDistributionDefinition := configuration.IncomeRecipients.Donations
	if len(donationDistributionDefinition) == 0 && configuration.IncomeRecipients.DonateBonds+configuration.IncomeRecipients.DonateFees > 0 { // inject default destination
		log.Trace("no donation destination found, donating to tezpay")
		donationDistributionDefinition = map[string]float64{
			constants.DEFAULT_DONATION_ADDRESS: 100,
		}
	}
	donationPayouts, err := getDistributionPayouts(enums.PAYOUT_KIND_DONATION, donationDistributionDefinition, ctx.StageData.DonateBondsAmount.Add(ctx.StageData.DonateFeesAmount), ctx, options)
	if err != nil {
		return ctx, fmt.Errorf("invalid donation distribution - %s", err.Error())
	}

	payouts := make([]common.PayoutRecipe, 0)
	payouts = append(payouts, delegatorPayouts...)
	payouts = append(payouts, bondsPayouts...)
	payouts = append(payouts, feesPayouts...)
	payouts = append(payouts, donationPayouts...)

	ctx.StageData.Payouts = payouts
	ctx.StageData.PaidDelegators = len(lo.Filter(delegatorPayouts, func(recipe common.PayoutRecipe, _ int) bool {
		return recipe.Kind != enums.PAYOUT_KIND_INVALID
	}))

	return ctx, nil
}
