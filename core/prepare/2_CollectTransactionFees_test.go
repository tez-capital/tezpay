package prepare

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/test/mock"
	"github.com/trilitech/tzgo/tezos"
)

var (
	payoutRecipes = []common.AccumulatedPayoutRecipe{
		{
			PayoutRecipe: common.PayoutRecipe{
				Delegator: mock.GetRandomAddress(),
				Recipient: mock.GetRandomAddress(),
				Amount:    tezos.NewZ(10000000),
				TxKind:    enums.PAYOUT_TX_KIND_TEZ,
				IsValid:   true,
			},
		},
		{
			PayoutRecipe: common.PayoutRecipe{
				Delegator: mock.GetRandomAddress(),
				Recipient: mock.GetRandomAddress(),
				Amount:    tezos.NewZ(20000000),
				TxKind:    enums.PAYOUT_TX_KIND_TEZ,
				IsValid:   true,
			},
		},
	}
)

func getRecipes() []*common.AccumulatedPayoutRecipe {
	result := []*common.AccumulatedPayoutRecipe{}
	for _, recipe := range payoutRecipes {
		result = append(result, &recipe)
	}
	return result
}

func TestCollectTransactionFees(t *testing.T) {
	var result *PayoutPrepareContext
	var err error
	config := configuration.GetDefaultRuntimeConfiguration()
	collector := mock.InitSimpleCollector()
	signer := mock.InitSimpleSigner()

	assert := assert.New(t)
	ctx := &PayoutPrepareContext{
		PreparePayoutsEngineContext: *common.NewPreparePayoutsEngineContext(collector, signer, nil, func(msg string) {}),
		StageData:                   &StageData{AccumulatedValidPayouts: getRecipes()},
		configuration:               &config,

		logger: slog.Default(),
	}

	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 0,
		StorageBurn:    0,
		UsedMilliGas:   2000000,
	})
	result, err = CollectTransactionFees(ctx, &common.PreparePayoutsOptions{})

	assert.Nil(err)
	for i, v := range result.StageData.AccumulatedValidPayouts {
		fmt.Println(collector.GetExpectedTxCosts())
		assert.LessOrEqual(v.Amount.Int64()-constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutRecipes[i].Amount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.Amount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutRecipes[i].Amount.Int64()-collector.GetExpectedTxCosts())
	}

	t.Log("check allocation burn")
	ctx.StageData.AccumulatedValidPayouts = getRecipes()
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   2000000,
	})
	result, err = CollectTransactionFees(ctx, &common.PreparePayoutsOptions{})

	assert.Nil(err)
	for i, v := range result.StageData.AccumulatedValidPayouts {
		assert.LessOrEqual(v.Amount.Int64()-constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutRecipes[i].Amount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.Amount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutRecipes[i].Amount.Int64()-collector.GetExpectedTxCosts())
	}

	t.Log("check storage burn")
	ctx.StageData.AccumulatedValidPayouts = getRecipes()
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
	})
	result, err = CollectTransactionFees(ctx, &common.PreparePayoutsOptions{})

	assert.Nil(err)
	for i, v := range result.StageData.AccumulatedValidPayouts {
		assert.LessOrEqual(v.Amount.Int64()-constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutRecipes[i].Amount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.Amount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutRecipes[i].Amount.Int64()-collector.GetExpectedTxCosts())
	}

	t.Log("check paying tx fee")
	ctx.StageData.AccumulatedValidPayouts = getRecipes()
	ctx.configuration.PayoutConfiguration.IsPayingTxFee = true
	ctx.configuration.PayoutConfiguration.IsPayingAllocationTxFee = false
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
	})
	result, _ = CollectTransactionFees(ctx, &common.PreparePayoutsOptions{})
	for i, v := range result.StageData.AccumulatedValidPayouts {
		assert.Equal(v.Amount.Int64(), payoutRecipes[i].Amount.Int64()-1000 /*allocation fee*/)
	}

	t.Log("check not paying tx & allocation fee")
	ctx.StageData.AccumulatedValidPayouts = getRecipes()
	ctx.configuration.PayoutConfiguration.IsPayingTxFee = true
	ctx.configuration.PayoutConfiguration.IsPayingAllocationTxFee = true
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
	})
	result, _ = CollectTransactionFees(ctx, &common.PreparePayoutsOptions{})
	for i, v := range result.StageData.AccumulatedValidPayouts {
		assert.Equal(v.Amount.Int64(), payoutRecipes[i].Amount.Int64())
	}

	t.Log("check per op estimate")
	ctx.StageData.AccumulatedValidPayouts = getRecipes()
	ctx.configuration.PayoutConfiguration.IsPayingTxFee = false
	ctx.configuration.PayoutConfiguration.IsPayingAllocationTxFee = false
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
		SingleOnly:     true,
	})
	result, err = CollectTransactionFees(ctx, &common.PreparePayoutsOptions{})

	assert.Nil(err)
	for i, v := range result.StageData.AccumulatedValidPayouts {
		assert.LessOrEqual(v.Amount.Int64()-constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutRecipes[i].Amount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.Amount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutRecipes[i].Amount.Int64()-collector.GetExpectedTxCosts())
	}

	t.Log("check batching")
	ctx.StageData.AccumulatedValidPayouts = getRecipes()
	ctx.configuration.PayoutConfiguration.IsPayingTxFee = false
	ctx.configuration.PayoutConfiguration.IsPayingAllocationTxFee = false
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
		SingleOnly:     true,
	})
	ops := []*common.AccumulatedPayoutRecipe{}
	for len(ops) < constants.DEFAULT_SIMULATION_TX_BATCH_SIZE*2.5 {
		ops = append(ops, getRecipes()...)
	}

	ctx.StageData.AccumulatedValidPayouts = ops
	result, err = CollectTransactionFees(ctx, &common.PreparePayoutsOptions{})
	assert.Nil(err)
	assert.Equal(len(ops), len(result.StageData.AccumulatedValidPayouts))

	ctx.StageData.AccumulatedValidPayouts = getRecipes()

	t.Log("fail estimate")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
		SingleOnly:     true,
		FailWithError:  errors.New("failed estimate"),
	})
	ctx.StageData.AccumulatedValidPayouts = getRecipes()
	result, _ = CollectTransactionFees(ctx, &common.PreparePayoutsOptions{})
	for _, v := range result.StageData.AccumulatedValidPayouts {
		assert.Equal(v.IsValid, false)
		assert.Equal(v.Note, enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS)
	}

	t.Log("failed receipt")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn:       1000,
		StorageBurn:          0,
		UsedMilliGas:         1000000,
		SingleOnly:           true,
		FailWithReceiptError: errors.New("failed receipt"),
	})
	ctx.StageData.AccumulatedValidPayouts = getRecipes()
	result, _ = CollectTransactionFees(ctx, &common.PreparePayoutsOptions{})
	for _, v := range result.StageData.AccumulatedValidPayouts {
		assert.Equal(v.IsValid, false)
		assert.Equal(v.Note, enums.INVALID_FAILED_TO_ESTIMATE_TX_COSTS)
	}

	t.Log("test partial panic")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn:   1000,
		StorageBurn:      0,
		UsedMilliGas:     1000000,
		SingleOnly:       false,
		ReturnOnlyNCosts: 1,
	})
	ctx.StageData.AccumulatedValidPayouts = getRecipes()
	assert.Panics(func() {
		_, err := CollectTransactionFees(ctx, &common.PreparePayoutsOptions{})
		t.Log(err)
	})
}
