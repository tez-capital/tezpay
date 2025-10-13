package common

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

func TestToReportsSuccessfulResult(t *testing.T) {
	assert := assert.New(t)

	payoutRecipe := &PayoutRecipe{
		Baker:            getRandomAddress(),
		Delegator:        getRandomAddress(),
		Cycle:            1,
		Recipient:        getRandomAddress(),
		Kind:             enums.PAYOUT_KIND_DELEGATOR_REWARD,
		TxKind:           enums.PAYOUT_TX_KIND_TEZ,
		FATokenId:        tezos.NewZ(1),
		FAContract:       getRandomAddress(),
		FAAlias:          "aaa",
		FADecimals:       2,
		DelegatedBalance: tezos.NewZ(1000000),
		StakedBalance:    tezos.NewZ(2000000),
		Amount:           tezos.NewZ(3000000),
		FeeRate:          5,
		Fee:              tezos.NewZ(4000000),
		TxFee:            10,
		Note:             "aaa",
		IsValid:          false,
	}

	accumulated := payoutRecipe.AsAccumulated()

	result := NewSuccessBatchResult([]*AccumulatedPayoutRecipe{accumulated}, tezos.ZeroOpHash)

	reports := result.ToIndividualReports()
	assert.Len(reports, 1)

	reportFromTheRecipe := payoutRecipe.ToPayoutReport()
	reportFromTheRecipe.IsSuccess = true // override success as we are comparing against a successful batch result

	reportTime := time.Now() // override time
	reportFromTheRecipe.Timestamp = reportTime
	reports[0].Timestamp = reportTime

	a, _ := json.Marshal(reportFromTheRecipe)
	b, _ := json.Marshal(reports[0])
	assert.Equal(a, b)
}

func TestToReportsFailedResult(t *testing.T) {
	assert := assert.New(t)

	payoutRecipe := &PayoutRecipe{
		Baker:            getRandomAddress(),
		Delegator:        getRandomAddress(),
		Cycle:            1,
		Recipient:        getRandomAddress(),
		Kind:             enums.PAYOUT_KIND_DELEGATOR_REWARD,
		TxKind:           enums.PAYOUT_TX_KIND_TEZ,
		FATokenId:        tezos.NewZ(1),
		FAContract:       getRandomAddress(),
		FAAlias:          "aaa",
		FADecimals:       2,
		DelegatedBalance: tezos.NewZ(1000000),
		StakedBalance:    tezos.NewZ(2000000),
		Amount:           tezos.NewZ(3000000),
		FeeRate:          5,
		Fee:              tezos.NewZ(4000000),
		TxFee:            10,
		Note:             "aaa",
		IsValid:          false,
	}

	accumulated := payoutRecipe.AsAccumulated()

	err := errors.New("test")
	result := NewFailedBatchResult([]*AccumulatedPayoutRecipe{accumulated}, err)

	reports := result.ToIndividualReports()
	assert.Len(reports, 1)

	reportFromTheRecipe := payoutRecipe.ToPayoutReport()
	reportFromTheRecipe.Note = err.Error() // override error as we are comparing against a failed batch result

	reportTime := time.Now() // override time
	reportFromTheRecipe.Timestamp = reportTime
	reports[0].Timestamp = reportTime

	a, _ := json.Marshal(reportFromTheRecipe)
	b, _ := json.Marshal(reports[0])
	assert.Equal(a, b)
}
