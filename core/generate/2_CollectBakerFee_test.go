package generate

import (
	"log/slog"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/test/mock"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/tezos"
)

var (
	payoutCandidatesWithBondAmount = []PayoutCandidateWithBondAmount{
		{
			PayoutCandidate: PayoutCandidate{
				Source:    mock.GetRandomAddress(),
				Recipient: mock.GetRandomAddress(),
				FeeRate:   0.05,
			},
			BondsAmount: tezos.NewZ(10000000),
			TxKind:      enums.PAYOUT_TX_KIND_TEZ,
		},
		{
			PayoutCandidate: PayoutCandidate{
				Source:    mock.GetRandomAddress(),
				Recipient: mock.GetRandomAddress(),
				FeeRate:   0.05,
			},
			BondsAmount: tezos.NewZ(20000000),
			TxKind:      enums.PAYOUT_TX_KIND_TEZ,
		},
		{
			PayoutCandidate: PayoutCandidate{
				Source:    mock.GetRandomAddress(),
				Recipient: mock.GetRandomAddress(),
				FeeRate:   0.05,
			},
			BondsAmount: tezos.NewZ(20000000),
			TxKind:      enums.PAYOUT_TX_KIND_FA1_2,
		},
	}
)

func adjustFee(ctx *PayoutGenerationContext, fee float64) {
	for i := range ctx.StageData.PayoutCandidatesWithBondAmount {
		ctx.StageData.PayoutCandidatesWithBondAmount[i].FeeRate = fee
	}
}

func TestCollectBakerFees(t *testing.T) {
	assert := assert.New(t)

	ctx := &PayoutGenerationContext{
		GeneratePayoutsEngineContext: *common.NewGeneratePayoutsEngines(collector, nil, nil),
		StageData:                    &StageData{PayoutCandidatesWithBondAmount: payoutCandidatesWithBondAmount},
		configuration:                &config,

		logger: slog.Default(),
	}

	t.Log("check 0 fee")
	adjustFee(ctx, 0)
	result, err := CollectBakerFee(ctx, &common.GeneratePayoutsOptions{})
	assert.Nil(err)
	assert.Equal(int64(0), result.StageData.BakerBondsAmount.Int64())
	for i, v := range result.StageData.PayoutCandidatesWithBondAmountAndFees {
		assert.Equal(payoutCandidatesWithBondAmount[i].BondsAmount.Int64(), v.BondsAmount.Int64())
		assert.Equal(int64(0), v.Fee.Int64())
	}

	t.Log("check 0.05 fee")
	feeRate := 0.05
	adjustFee(ctx, feeRate)
	result, err = CollectBakerFee(ctx, &common.GeneratePayoutsOptions{})
	assert.Nil(err)
	feesAmount := lo.Reduce(payoutCandidatesWithBondAmount, func(agg int64, v PayoutCandidateWithBondAmount, _ int) int64 {
		if v.TxKind != enums.PAYOUT_TX_KIND_TEZ {
			return agg
		}
		return agg + utils.GetZPortion(v.BondsAmount, feeRate).Int64()
	}, int64(0))
	assert.Equal(feesAmount, result.StageData.BakerFeesAmount.Int64())
	for i, v := range result.StageData.PayoutCandidatesWithBondAmountAndFees {
		if payoutCandidatesWithBondAmount[i].TxKind != enums.PAYOUT_TX_KIND_TEZ {
			continue
		}
		assert.Equal(utils.GetZPortion(payoutCandidatesWithBondAmount[i].BondsAmount, 1-feeRate).Int64(), v.BondsAmount.Int64())
		assert.Equal(utils.GetZPortion(payoutCandidatesWithBondAmount[i].BondsAmount, feeRate).Int64(), v.Fee.Int64())
	}

	t.Log("check donate")
	donationRate := float64(0.02)
	ctx.configuration.IncomeRecipients.DonateFees = donationRate
	result, err = CollectBakerFee(ctx, &common.GeneratePayoutsOptions{})
	assert.Nil(err)
	donateAmount := lo.Reduce(payoutCandidatesWithBondAmount, func(agg int64, v PayoutCandidateWithBondAmount, _ int) int64 {
		if v.TxKind != enums.PAYOUT_TX_KIND_TEZ {
			return agg
		}
		return agg + utils.GetZPortion(utils.GetZPortion(v.BondsAmount, feeRate), donationRate).Int64()
	}, int64(0))
	assert.Equal(donateAmount, result.StageData.DonateFeesAmount.Int64())
	for i, v := range result.StageData.PayoutCandidatesWithBondAmountAndFees {
		if payoutCandidatesWithBondAmount[i].TxKind != enums.PAYOUT_TX_KIND_TEZ {
			continue
		}
		assert.Equal(utils.GetZPortion(payoutCandidatesWithBondAmount[i].BondsAmount, 1-feeRate).Int64(), v.BondsAmount.Int64())
		assert.Equal(utils.GetZPortion(payoutCandidatesWithBondAmount[i].BondsAmount, feeRate).Int64(), v.Fee.Int64())
	}

	t.Log("check 1 fee")
	feeRate = 1
	adjustFee(ctx, feeRate)
	result, err = CollectBakerFee(ctx, &common.GeneratePayoutsOptions{})
	assert.Nil(err)
	collectedFee := tezos.Zero
	for _, v := range result.StageData.PayoutCandidatesWithBondAmountAndFees {
		if v.TxKind != enums.PAYOUT_TX_KIND_TEZ {
			continue
		}
		assert.True(v.IsInvalid)
		assert.Equal(v.InvalidBecause, enums.INVALID_PAYOUT_BELLOW_MINIMUM)
		collectedFee = collectedFee.Add(v.Fee)
	}
	totalBonds := lo.Reduce(ctx.StageData.PayoutCandidatesWithBondAmount, func(agg tezos.Z, v PayoutCandidateWithBondAmount, _ int) tezos.Z {
		if v.TxKind != enums.PAYOUT_TX_KIND_TEZ {
			return agg
		}
		return agg.Add(v.BondsAmount)
	}, tezos.Zero)

	assert.True(collectedFee.Equal(totalBonds))

	t.Log("invalidCandidates")
	ctx.StageData.PayoutCandidatesWithBondAmount = lo.Map(payoutCandidatesWithBondAmount, func(candidate PayoutCandidateWithBondAmount, index int) PayoutCandidateWithBondAmount {
		candidate.IsInvalid = true
		if index == 0 {
			candidate.InvalidBecause = enums.INVALID_DELEGATOR_EMPTIED
		} else if index == 1 {
			candidate.InvalidBecause = enums.INVALID_DELEGATOR_IGNORED
		}
		return candidate
	})
	result, err = CollectBakerFee(ctx, &common.GeneratePayoutsOptions{})
	assert.Nil(err)
	for index, v := range result.StageData.PayoutCandidatesWithBondAmountAndFees {
		assert.True(v.IsInvalid)
		if index == 0 {
			assert.Equal(v.InvalidBecause, enums.INVALID_DELEGATOR_EMPTIED)
		} else if index == 1 {
			assert.Equal(v.InvalidBecause, enums.INVALID_DELEGATOR_IGNORED)
		}
	}
}
