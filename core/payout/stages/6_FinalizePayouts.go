package stages

import (
	"fmt"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func getDistributionPayouts(kind enums.EPayoutKind, distributionDefinition map[string]float32, amount tezos.Z, ctx Context) ([]common.PayoutRecipe, error) {
	totalPercentage := lo.Reduce(lo.Values(distributionDefinition), func(agg float32, entry float32, _ int) float32 {
		return agg + entry
	}, float32(0))
	if totalPercentage > 100 {
		return []common.PayoutRecipe{}, fmt.Errorf("expects <= 100%% but got %f", totalPercentage)
	}
	i := 0
	result := make([]common.PayoutRecipe, len(distributionDefinition))
	for recipient, percentage := range distributionDefinition {
		recipient, err := tezos.ParseAddress(recipient)
		if err != nil {
			return []common.PayoutRecipe{}, err
		}

		recipientPortion := utils.GetZPortion(amount, percentage)
		if recipientPortion.IsZero() {
			continue
		}

		// simulate - because of batch spliting
		op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
		op.WithTransfer(recipient, recipientPortion.Int64())

		receipt, err := ctx.Collector.Simulate(op, ctx.PayoutKey)

		if err != nil {
			return []common.PayoutRecipe{}, err
		}
		costs := receipt.TotalCosts()

		result[i] = common.PayoutRecipe{
			Baker:            ctx.GetConfiguration().BakerPKH,
			Cycle:            ctx.Cycle,
			Kind:             kind,
			Delegator:        tezos.ZeroAddress,
			Recipient:        recipient,
			DelegatedBalance: tezos.Zero,
			Amount:           recipientPortion,
			FeeRate:          0,
			Fee:              tezos.Zero,
			OpLimits: &common.OpLimits{
				GasLimit:       costs.GasUsed + constants.GAS_LIMIT_BUFFER,
				StorageLimit:   utils.CalculateStorageLimit(costs),
				TransactionFee: utils.EstimateTransactionFee(op, receipt.Costs()),
			},
			Note:    "",
			IsValid: true,
		}
		i += 1
	}
	return result, nil
}

// injects bonds, fee and donation payments and finalizes Payouts
func finalizePayouts(ctx Context) (result Context, err error) {
	configuration := ctx.GetConfiguration()
	log.Debug("finalizing payouts")
	simulated := ctx.StageData.PayoutCandidatesSimulated

	delegatorPayouts := lo.Map(simulated, func(candidate PayoutCandidateSimulated, _ int) common.PayoutRecipe {
		return candidate.ToPayoutRecipe(ctx.GetConfiguration().BakerPKH, ctx.Cycle, enums.PAYOUT_KIND_DELEGATOR_REWARD)
	})

	// bonds
	bondsPayouts, err := getDistributionPayouts(enums.PAYOUT_KIND_BAKER_REWARD, configuration.IncomeRecipients.Bonds, ctx.StageData.BakerBondsAmount, ctx)
	if err != nil {
		return ctx, fmt.Errorf("invalid bonds distribution - %s", err.Error())
	}

	// fees
	feesPayouts, err := getDistributionPayouts(enums.PAYOUT_KIND_FEE_INCOME, configuration.IncomeRecipients.Fees, ctx.StageData.BakerFeesAmount, ctx)
	if err != nil {
		return ctx, fmt.Errorf("invalid fees distribution - %s", err.Error())
	}

	// donations
	donationDistributionDefinition := configuration.IncomeRecipients.Donations
	if len(donationDistributionDefinition) == 0 && configuration.IncomeRecipients.Donate > 0 { // inject default destination
		log.Trace("no donation destination found, donating to tezpay")
		donationDistributionDefinition = map[string]float32{
			constants.DEFAULT_DONATION_ADDRESS: 100,
		}
	}
	donationPayouts, err := getDistributionPayouts(enums.PAYOUT_KIND_DONATION, donationDistributionDefinition, ctx.StageData.DonateBondsAmount.Add(ctx.StageData.DonateFeesAmount), ctx)
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

var FinalizePayouts = WrapStage(finalizePayouts)
