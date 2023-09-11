package generate

import (
	"errors"
	"testing"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/test/mock"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

var (
	payoutCandidatesWithBondAmountAndFees = []PayoutCandidateWithBondAmountAndFee{
		{
			PayoutCandidateWithBondAmount: PayoutCandidateWithBondAmount{
				PayoutCandidate: PayoutCandidate{
					Source:    mock.GetRandomAddress(),
					Recipient: mock.GetRandomAddress(),
				},
				BondsAmount: tezos.NewZ(10000000),
				TxKind:      enums.PAYOUT_TX_KIND_TEZ,
			},
		},
		{
			PayoutCandidateWithBondAmount: PayoutCandidateWithBondAmount{
				PayoutCandidate: PayoutCandidate{
					Source:    mock.GetRandomAddress(),
					Recipient: mock.GetRandomAddress(),
				},
				BondsAmount: tezos.NewZ(20000000),
				TxKind:      enums.PAYOUT_TX_KIND_TEZ,
			},
		},
	}
	config    = configuration.GetDefaultRuntimeConfiguration()
	collector = mock.InitSimpleColletor()
)

func TestCollectTransactionFees(t *testing.T) {
	assert := assert.New(t)
	ctx := &PayoutGenerationContext{
		GeneratePayoutsEngineContext: *common.NewGeneratePayoutsEngines(collector, nil, nil),
		StageData:                    &StageData{PayoutCandidatesWithBondAmountAndFees: payoutCandidatesWithBondAmountAndFees},
		configuration:                &config,
	}

	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 0,
		StorageBurn:    0,
		UsedMilliGas:   2000000,
	})
	result, err := CollectTransactionFees(ctx, &common.GeneratePayoutsOptions{})

	assert.Nil(err)
	for i, v := range result.StageData.PayoutCandidatesSimulated {
		assert.LessOrEqual(v.BondsAmount.Int64()-constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.BondsAmount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
	}

	t.Log("check allocation burn")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   2000000,
	})
	result, err = CollectTransactionFees(ctx, &common.GeneratePayoutsOptions{})

	assert.Nil(err)
	for i, v := range result.StageData.PayoutCandidatesSimulated {
		assert.LessOrEqual(v.BondsAmount.Int64()-constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.BondsAmount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
	}

	t.Log("check storage burn")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
	})
	result, err = CollectTransactionFees(ctx, &common.GeneratePayoutsOptions{})

	assert.Nil(err)
	for i, v := range result.StageData.PayoutCandidatesSimulated {
		assert.LessOrEqual(v.BondsAmount.Int64()-constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.BondsAmount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
	}

	t.Log("chech paying tx fee")

	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = lo.Map(ctx.StageData.PayoutCandidatesWithBondAmountAndFees, func(v PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateWithBondAmountAndFee {
		v.IsBakerPayingTxFee = true
		v.IsBakerPayingAllocationTxFee = false
		return v
	})
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
	})
	result, _ = CollectTransactionFees(ctx, &common.GeneratePayoutsOptions{})
	for i, v := range result.StageData.PayoutCandidatesSimulated {
		assert.Equal(v.BondsAmount.Int64(), payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-1000 /*allocation fee*/)
	}

	t.Log("chech not paying tx & allocation fee")
	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = lo.Map(ctx.StageData.PayoutCandidatesWithBondAmountAndFees, func(v PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateWithBondAmountAndFee {
		v.IsBakerPayingTxFee = true
		v.IsBakerPayingAllocationTxFee = true
		return v
	})
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
	})
	result, _ = CollectTransactionFees(ctx, &common.GeneratePayoutsOptions{})
	for i, v := range result.StageData.PayoutCandidatesSimulated {
		assert.Equal(v.BondsAmount.Int64(), payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64())
	}

	t.Log("chech per op estimate")
	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = lo.Map(ctx.StageData.PayoutCandidatesWithBondAmountAndFees, func(v PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateWithBondAmountAndFee {
		v.IsBakerPayingTxFee = false
		v.IsBakerPayingAllocationTxFee = false
		return v
	})
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
		SingleOnly:     true,
	})
	result, err = CollectTransactionFees(ctx, &common.GeneratePayoutsOptions{})

	assert.Nil(err)
	for i, v := range result.StageData.PayoutCandidatesSimulated {
		assert.LessOrEqual(v.BondsAmount.Int64()-constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.BondsAmount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
	}

	t.Log("check batching")
	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = lo.Map(ctx.StageData.PayoutCandidatesWithBondAmountAndFees, func(v PayoutCandidateWithBondAmountAndFee, _ int) PayoutCandidateWithBondAmountAndFee {
		v.IsBakerPayingTxFee = false
		v.IsBakerPayingAllocationTxFee = false
		return v
	})
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
		SingleOnly:     true,
	})
	ops := []PayoutCandidateWithBondAmountAndFee{}
	for len(ops) < TX_BATCH_CAPACITY*2.5 {
		ops = append(ops, payoutCandidatesWithBondAmountAndFees...)
	}

	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = ops
	result, _ = CollectTransactionFees(ctx, &common.GeneratePayoutsOptions{})
	assert.Equal(len(result.StageData.PayoutCandidatesSimulated), len(ops))

	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = payoutCandidatesWithBondAmountAndFees

	t.Log("fail estimate")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
		SingleOnly:     true,
		FailWithError:  errors.New("failed estimate"),
	})
	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = payoutCandidatesWithBondAmountAndFees
	result, _ = CollectTransactionFees(ctx, &common.GeneratePayoutsOptions{})
	for _, v := range result.StageData.PayoutCandidatesSimulated {
		assert.Equal(v.IsInvalid, true)
		assert.Equal(v.InvalidBecause, enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS)
	}

	t.Log("failed receipt")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn:       1000,
		StorageBurn:          0,
		UsedMilliGas:         1000000,
		SingleOnly:           true,
		FailWithReceiptError: errors.New("failed receipt"),
	})
	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = payoutCandidatesWithBondAmountAndFees
	result, _ = CollectTransactionFees(ctx, &common.GeneratePayoutsOptions{})
	for _, v := range result.StageData.PayoutCandidatesSimulated {
		assert.Equal(v.IsInvalid, true)
		assert.Equal(v.InvalidBecause, enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS)
	}

	t.Log("test partial panic")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn:   1000,
		StorageBurn:      0,
		UsedMilliGas:     1000000,
		SingleOnly:       false,
		ReturnOnlyNCosts: 1,
	})
	ctx.StageData.PayoutCandidatesWithBondAmountAndFees = payoutCandidatesWithBondAmountAndFees
	assert.Panics(func() {
		_, err := CollectTransactionFees(ctx, &common.GeneratePayoutsOptions{})
		t.Log(err)
	})
}
