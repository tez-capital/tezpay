package stages

import (
	"testing"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/test/mock"
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
			},
		},
		{
			PayoutCandidateWithBondAmount: PayoutCandidateWithBondAmount{
				PayoutCandidate: PayoutCandidate{
					Source:    mock.GetRandomAddress(),
					Recipient: mock.GetRandomAddress(),
				},
				BondsAmount: tezos.NewZ(20000000),
			},
		},
	}
	config    = configuration.GetDefaultRuntimeConfiguration()
	collector = mock.InitSimpleColletor()
	ctx       = Context{
		StageData:     StageData{PayoutCandidatesWithBondAmountAndFees: payoutCandidatesWithBondAmountAndFees},
		Collector:     collector,
		configuration: &config,
	}
)

func TestCollectTransactionFees(t *testing.T) {
	assert := assert.New(t)

	t.Log("check gas usage")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 0,
		StorageBurn:    0,
		UsedMilliGas:   2000000,
	})
	result := CollectTransactionFees(WrappedStageResult{Ctx: ctx, Err: nil})

	assert.Nil(result.Err)
	for i, v := range result.Ctx.StageData.PayoutCandidatesSimulated {
		assert.LessOrEqual(v.BondsAmount.Int64(), payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.BondsAmount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
	}

	t.Log("check allocation burn")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   2000000,
	})
	result = CollectTransactionFees(WrappedStageResult{Ctx: ctx, Err: nil})

	assert.Nil(result.Err)
	for i, v := range result.Ctx.StageData.PayoutCandidatesSimulated {
		assert.LessOrEqual(v.BondsAmount.Int64(), payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.BondsAmount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
	}

	t.Log("check storage burn")
	collector.SetOpts(&mock.SimpleCollectorOpts{
		AllocationBurn: 1000,
		StorageBurn:    0,
		UsedMilliGas:   1000000,
	})
	result = CollectTransactionFees(WrappedStageResult{Ctx: ctx, Err: nil})

	assert.Nil(result.Err)
	for i, v := range result.Ctx.StageData.PayoutCandidatesSimulated {
		assert.LessOrEqual(v.BondsAmount.Int64(), payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
		assert.GreaterOrEqual(v.BondsAmount.Int64()+constants.TEST_MUTEZ_DEVIATION_TOLERANCE, payoutCandidatesWithBondAmountAndFees[i].BondsAmount.Int64()-collector.GetExpectedTxCosts())
	}
}
