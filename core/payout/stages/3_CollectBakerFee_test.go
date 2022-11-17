package stages

import (
	"testing"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/test/mock"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
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
		},
		{
			PayoutCandidate: PayoutCandidate{
				Source:    mock.GetRandomAddress(),
				Recipient: mock.GetRandomAddress(),
				FeeRate:   0.05,
			},
			BondsAmount: tezos.NewZ(20000000),
		},
	}
)

func adjustFee(ctx Context, fee float64) {
	for i := range ctx.StageData.PayoutCandidatesWithBondAmount {
		ctx.StageData.PayoutCandidatesWithBondAmount[i].FeeRate = fee
	}
}

func TestCollectBakerFees(t *testing.T) {
	assert := assert.New(t)

	ctx := Context{
		StageData:     StageData{PayoutCandidatesWithBondAmount: payoutCandidatesWithBondAmount},
		Collector:     collector,
		configuration: &config,
	}

	t.Log("check 0 fee")
	adjustFee(ctx, 0)
	result := CollectBakerFee(WrappedStageResult{Ctx: ctx, Err: nil})
	assert.Nil(result.Err)
	assert.Equal(int64(0), result.Ctx.StageData.BakerBondsAmount.Int64())
	for i, v := range result.Ctx.StageData.PayoutCandidatesWithBondAmountAndFees {
		assert.Equal(payoutCandidatesWithBondAmount[i].BondsAmount.Int64(), v.BondsAmount.Int64())
		assert.Equal(int64(0), v.Fee.Int64())
	}

	t.Log("check 0.05 fee")
	feeRate := 0.05
	adjustFee(ctx, feeRate)
	result = CollectBakerFee(WrappedStageResult{Ctx: ctx, Err: nil})
	assert.Nil(result.Err)
	feesAmount := lo.Reduce(payoutCandidatesWithBondAmount, func(agg int64, v PayoutCandidateWithBondAmount, _ int) int64 {
		return agg + utils.GetZPortion(v.BondsAmount, feeRate).Int64()
	}, int64(0))
	assert.Equal(feesAmount, result.Ctx.StageData.BakerFeesAmount.Int64())
	for i, v := range result.Ctx.StageData.PayoutCandidatesWithBondAmountAndFees {
		assert.Equal(utils.GetZPortion(payoutCandidatesWithBondAmount[i].BondsAmount, 1-feeRate).Int64(), v.BondsAmount.Int64())
		assert.Equal(utils.GetZPortion(payoutCandidatesWithBondAmount[i].BondsAmount, feeRate).Int64(), v.Fee.Int64())
	}

	t.Log("check donate")
	donationRate := float64(0.02)
	ctx.configuration.IncomeRecipients.Donate = donationRate
	result = CollectBakerFee(WrappedStageResult{Ctx: ctx, Err: nil})
	assert.Nil(result.Err)
	donateAmount := lo.Reduce(payoutCandidatesWithBondAmount, func(agg int64, v PayoutCandidateWithBondAmount, _ int) int64 {
		return agg + utils.GetZPortion(utils.GetZPortion(v.BondsAmount, feeRate), donationRate).Int64()
	}, int64(0))
	assert.Equal(donateAmount, result.Ctx.StageData.DonateFeesAmount.Int64())
	for i, v := range result.Ctx.StageData.PayoutCandidatesWithBondAmountAndFees {
		assert.Equal(utils.GetZPortion(payoutCandidatesWithBondAmount[i].BondsAmount, 1-feeRate).Int64(), v.BondsAmount.Int64())
		assert.Equal(utils.GetZPortion(payoutCandidatesWithBondAmount[i].BondsAmount, feeRate).Int64(), v.Fee.Int64())
	}

	t.Log("check 1 fee")
	feeRate = 1
	adjustFee(ctx, feeRate)
	result = CollectBakerFee(WrappedStageResult{Ctx: ctx, Err: nil})
	assert.Nil(result.Err)
	for _, v := range result.Ctx.StageData.PayoutCandidatesWithBondAmountAndFees {
		assert.True(v.IsInvalid)
		assert.Equal(v.InvalidBecause, enums.INVALID_PAYOUT_BELLOW_MINIMUM)
	}

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
	result = CollectBakerFee(WrappedStageResult{Ctx: ctx, Err: nil})
	assert.Nil(result.Err)
	for index, v := range result.Ctx.StageData.PayoutCandidatesWithBondAmountAndFees {
		assert.True(v.IsInvalid)
		if index == 0 {
			assert.Equal(v.InvalidBecause, enums.INVALID_DELEGATOR_EMPTIED)
		} else if index == 1 {
			assert.Equal(v.InvalidBecause, enums.INVALID_DELEGATOR_IGNORED)
		}
	}
}
