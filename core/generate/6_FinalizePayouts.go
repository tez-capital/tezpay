package generate

import (
	"fmt"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/core/estimate"
	"github.com/trilitech/tzgo/tezos"

	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/tez-capital/tezpay/utils"
)

func getDistributionPayouts(kind enums.EPayoutKind, distributionDefinition map[string]float64, amount tezos.Z, ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) ([]common.PayoutRecipe, error) {
	totalPercentage := lo.Reduce(lo.Values(distributionDefinition), func(agg float64, entry float64, _ int) float64 {
		return agg + entry
	}, float64(0))

	if totalPercentage > 100 {
		return []common.PayoutRecipe{}, fmt.Errorf("expects <= 100%% but only has %f", totalPercentage)
	}

	valid := make([]common.PayoutRecipe, 0, len(distributionDefinition))
	invalid := make([]common.PayoutRecipe, 0, len(distributionDefinition))
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
			invalid = append(invalid, recipe)
			continue
		}
		recipe.Recipient = recipient
		if recipient.Equal(ctx.PayoutKey.Address()) {
			recipe.IsValid = false
			recipe.Note = string(enums.INVALID_RECIPIENT_TARGETS_PAYOUT)
			invalid = append(invalid, recipe)
			continue
		}

		recipientPortion := utils.GetZPortion(amount, portion)
		recipe.Amount = recipientPortion
		if recipientPortion.IsZero() || recipientPortion.IsNeg() {
			recipe.IsValid = false
			recipe.Note = string(enums.INVALID_PAYOUT_ZERO)
			invalid = append(invalid, recipe)
			continue
		}

		// // simulate - because of batch spliting
		// op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
		// op.WithTTL(constants.MAX_OPERATION_TTL)
		// op.WithTransfer(tezos.BurnAddress, 1)
		// op.WithTransfer(recipient, recipientPortion.Int64())
		// op.WithTransfer(tezos.BurnAddress, 1)

		// receipt, err := ctx.GetCollector().Simulate(op, ctx.PayoutKey)

		// if err != nil {
		// 	return []common.PayoutRecipe{}, err
		// }
		// costs := receipt.Costs()
		// if len(costs) < 3 {
		// 	return []common.PayoutRecipe{}, fmt.Errorf("invalid costs length, cannot estimate")
		// }

		// // we use entire serialization cost even with two burn txs, it is used as some offset to avoid exhaustion
		// serializationGas := (costs[0].GasUsed - costs[len(costs)-1].GasUsed) - ctx.StageData.BatchMetadataDeserializationGasLimit
		// op.Contents = op.Contents[1 : len(op.Contents)-1]
		// costs = costs[1 : len(costs)-1]
		// cost := costs[0]

		// common.InjectLimits(op, []tezos.Limits{{
		// 	GasLimit:     cost.GasUsed + ctx.configuration.PayoutConfiguration.TxGasLimitBuffer,
		// 	StorageLimit: utils.CalculateStorageLimit(cost),
		// 	Fee:          cost.Fee,
		// }})

		// feeBuffer := ctx.configuration.PayoutConfiguration.TxFeeBuffer
		// if recipient.IsContract() {
		// 	feeBuffer = ctx.configuration.PayoutConfiguration.KtTxFeeBuffer
		// }

		// totalOpGasUsed := cost.GasUsed + serializationGas +
		// 	ctx.configuration.PayoutConfiguration.TxGasLimitBuffer + // buffer for gas limit
		// 	ctx.StageData.BatchMetadataDeserializationGasLimit + // potential gas used for deserialization if only one tx in batch
		// 	ctx.configuration.PayoutConfiguration.TxDeserializationGasBuffer // buffer for deserialization gas limit

		// recipe.OpLimits = &common.OpLimits{
		// 	GasLimit:                cost.GasUsed + ctx.configuration.PayoutConfiguration.TxGasLimitBuffer,
		// 	StorageLimit:            utils.CalculateStorageLimit(cost),
		// 	TransactionFee:          utils.EstimateTransactionFee(op, []int64{totalOpGasUsed}, feeBuffer),
		// 	DeserializationGasLimit: serializationGas + ctx.configuration.PayoutConfiguration.TxDeserializationGasBuffer,
		// }
		valid = append(valid, recipe)
	}

	estimateContext := &estimate.EstimationContext{
		PayoutKey:                            ctx.PayoutKey,
		Collector:                            ctx.GetCollector(),
		Configuration:                        ctx.GetConfiguration(),
		BatchMetadataDeserializationGasLimit: ctx.StageData.BatchMetadataDeserializationGasLimit,
	}

	all := lo.Map(estimate.EstimateTransactionFees(utils.MapToPointers(valid), estimateContext), func(result estimate.EstimateResult[*common.PayoutRecipe], _ int) common.PayoutRecipe {
		if result.Error != nil {
			log.Warnf("failed to estimate tx costs to '%s' (delegator: '%s', amount %d, kind '%s')\nerror: %s", result.Transaction.Recipient, ctx.PayoutKey.Address(), result.Transaction.Amount.Int64(), result.Transaction.TxKind, result.Error.Error())
			result.Transaction.IsValid = false
			result.Transaction.Note = string(enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS)
		}
		result.Transaction.OpLimits = result.Result
		return *result.Transaction
	})
	all = append(all, invalid...)
	return all, nil
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
		log.Trace("no donation destination found, donating to tez.capital")
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
