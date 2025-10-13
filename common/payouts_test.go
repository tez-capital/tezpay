package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

func getRandomAddress() tezos.Address {
	k, _ := tezos.GenerateKey(tezos.KeyTypeEd25519)
	return k.Address()
}

func TestPayoutRecipeToAccumulatedIdentifierMatches(t *testing.T) {
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

	assert.Equal(payoutRecipe.GetIdentifier(), accumulated.GetIdentifier())
}

func TestPayoutRecipeToAccumulated(t *testing.T) {
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
	assert.Len(accumulated.Recipes, 1)

	a, _ := json.Marshal(payoutRecipe)
	b, _ := json.Marshal(accumulated.Recipes[0])
	assert.Equal(a, b)

	assert.True(accumulated.Delegator.Equal(payoutRecipe.Delegator))
	assert.Equal(accumulated.Cycle, payoutRecipe.Cycle)
	assert.True(accumulated.Recipient.Equal(payoutRecipe.Recipient))
	assert.Equal(accumulated.Kind, payoutRecipe.Kind)
	assert.Equal(accumulated.TxKind, payoutRecipe.TxKind)
	assert.True(accumulated.FATokenId.Equal(payoutRecipe.FATokenId))
	assert.True(accumulated.FAContract.Equal(payoutRecipe.FAContract))
	assert.Equal(accumulated.IsValid, payoutRecipe.IsValid)
	assert.Equal(accumulated.Note, payoutRecipe.Note)
}

func Test_AccumulatedAddRecipeSame(t *testing.T) {
	assert := assert.New(t)

	baker := getRandomAddress()
	delegator := getRandomAddress()
	recipient := getRandomAddress()
	faContract := getRandomAddress()

	payoutRecipe := &PayoutRecipe{
		Baker:            baker,
		Delegator:        delegator,
		Cycle:            1,
		Recipient:        recipient,
		Kind:             enums.PAYOUT_KIND_DELEGATOR_REWARD,
		TxKind:           enums.PAYOUT_TX_KIND_TEZ,
		FATokenId:        tezos.NewZ(1),
		FAContract:       faContract,
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
	payoutRecipe2 := &PayoutRecipe{
		Baker:            baker,
		Delegator:        delegator,
		Cycle:            1,
		Recipient:        recipient,
		Kind:             enums.PAYOUT_KIND_DELEGATOR_REWARD,
		TxKind:           enums.PAYOUT_TX_KIND_TEZ,
		FATokenId:        tezos.NewZ(1),
		FAContract:       faContract,
		FAAlias:          "aaa",
		FADecimals:       2,
		DelegatedBalance: tezos.NewZ(2000000),
		StakedBalance:    tezos.NewZ(3000000),
		Amount:           tezos.NewZ(4000000),
		FeeRate:          5,
		Fee:              tezos.NewZ(5000000),
		TxFee:            10,
		Note:             "aaa",
		IsValid:          false,
	}

	accumulated := payoutRecipe.AsAccumulated()
	_, err := accumulated.Add(payoutRecipe2)
	assert.NoError(err)

	assert.Len(accumulated.Recipes, 2)
}

func Test_AccumulatedAddRecipeDifferent(t *testing.T) {
	assert := assert.New(t)

	baker := getRandomAddress()
	delegator := getRandomAddress()
	recipient := getRandomAddress()
	faContract := getRandomAddress()

	getPayoutRecipe1 := func() *PayoutRecipe {
		return &PayoutRecipe{
			Baker:            baker,
			Delegator:        delegator,
			Cycle:            1,
			Recipient:        recipient,
			Kind:             enums.PAYOUT_KIND_DELEGATOR_REWARD,
			TxKind:           enums.PAYOUT_TX_KIND_TEZ,
			FATokenId:        tezos.NewZ(1),
			FAContract:       faContract,
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
	}

	getPayoutRecipe2 := func() *PayoutRecipe {
		return &PayoutRecipe{
			Baker:            baker,
			Delegator:        delegator,
			Cycle:            1,
			Recipient:        recipient,
			Kind:             enums.PAYOUT_KIND_DELEGATOR_REWARD,
			TxKind:           enums.PAYOUT_TX_KIND_TEZ,
			FATokenId:        tezos.NewZ(1),
			FAContract:       faContract,
			FAAlias:          "aaa",
			FADecimals:       2,
			DelegatedBalance: tezos.NewZ(2000000),
			StakedBalance:    tezos.NewZ(3000000),
			Amount:           tezos.NewZ(4000000),
			FeeRate:          5,
			Fee:              tezos.NewZ(5000000),
			TxFee:            10,
			Note:             "aaa",
			IsValid:          false,
		}
	}

	var accumulated *AccumulatedPayoutRecipe
	var toAdd *PayoutRecipe
	accumulated = getPayoutRecipe1().AsAccumulated()
	toAdd = getPayoutRecipe2()
	_, err := accumulated.Add(toAdd)
	assert.NoError(err)
	assert.Len(accumulated.Recipes, 2)

	toAdd = getPayoutRecipe2()
	toAdd.Recipient = getRandomAddress()
	_, err = accumulated.Add(toAdd)
	assert.Error(err)
	assert.ErrorContains(err, "cannot add different recipients")

	toAdd = getPayoutRecipe2()
	toAdd.Delegator = getRandomAddress()
	_, err = accumulated.Add(toAdd)
	assert.Error(err)
	assert.ErrorContains(err, "cannot add different delegators")

	toAdd = getPayoutRecipe2()
	toAdd.Kind = enums.PAYOUT_KIND_DONATION
	_, err = accumulated.Add(toAdd)
	assert.Error(err)
	assert.ErrorContains(err, "cannot add different kinds")

	toAdd = getPayoutRecipe2()
	toAdd.TxKind = enums.PAYOUT_TX_KIND_FA2
	_, err = accumulated.Add(toAdd)
	assert.Error(err)
	assert.ErrorContains(err, "cannot add different tx kinds")

	toAdd = getPayoutRecipe2()
	toAdd.FATokenId = tezos.NewZ(2)
	_, err = accumulated.Add(toAdd)
	assert.Error(err)
	assert.ErrorContains(err, "cannot add different FA token ids")

	toAdd = getPayoutRecipe2()
	toAdd.FAContract = getRandomAddress()
	_, err = accumulated.Add(toAdd)
	assert.Error(err)
	assert.ErrorContains(err, "cannot add different FA contracts")

	toAdd = getPayoutRecipe2()
	toAdd.IsValid = true
	_, err = accumulated.Add(toAdd)
	assert.Error(err)
	assert.ErrorContains(err, "cannot add different validity states")
}

func TestAddTxFee(t *testing.T) {
	assert := assert.New(t)

	baker := getRandomAddress()
	delegator := getRandomAddress()
	recipient := getRandomAddress()
	faContract := getRandomAddress()

	payoutRecipe := &PayoutRecipe{
		Baker:            baker,
		Delegator:        delegator,
		Cycle:            1,
		Recipient:        recipient,
		Kind:             enums.PAYOUT_KIND_DELEGATOR_REWARD,
		TxKind:           enums.PAYOUT_TX_KIND_TEZ,
		FATokenId:        tezos.NewZ(1),
		FAContract:       faContract,
		FAAlias:          "aaa",
		FADecimals:       2,
		DelegatedBalance: tezos.NewZ(1000000),
		StakedBalance:    tezos.NewZ(2000000),
		Amount:           tezos.NewZ(3000000),
		FeeRate:          5,
		Fee:              tezos.NewZ(4000000),
		TxFee:            0,
		Note:             "aaa",
		IsValid:          false,
	}
	payoutRecipe2 := &PayoutRecipe{
		Baker:            baker,
		Delegator:        delegator,
		Cycle:            1,
		Recipient:        recipient,
		Kind:             enums.PAYOUT_KIND_DELEGATOR_REWARD,
		TxKind:           enums.PAYOUT_TX_KIND_TEZ,
		FATokenId:        tezos.NewZ(1),
		FAContract:       faContract,
		FAAlias:          "aaa",
		FADecimals:       2,
		DelegatedBalance: tezos.NewZ(2000000),
		StakedBalance:    tezos.NewZ(3000000),
		Amount:           tezos.NewZ(4000000),
		FeeRate:          5,
		Fee:              tezos.NewZ(5000000),
		TxFee:            0,
		Note:             "aaa",
		IsValid:          false,
	}

	accumulated := payoutRecipe.AsAccumulated()
	_, err := accumulated.Add(payoutRecipe2)
	assert.NoError(err)
	assert.Len(accumulated.Recipes, 2)
	assert.Equal(accumulated.GetAmount(), tezos.NewZ(7000000))

	accumulated.AddTxFee64(1000, false)
	assert.Equal(accumulated.GetAmount(), tezos.NewZ(7000000))
	assert.Equal(accumulated.GetTxFee(), int64(1000))

	accumulated.AddTxFee64(2000, true)
	assert.Equal(accumulated.GetAmount(), tezos.NewZ(7000000).Sub(tezos.NewZ(2000)))
	assert.Equal(accumulated.GetTxFee(), int64(3000))

	accumulated.AddTxFee64(7000000, true)
	assert.Equal(accumulated.GetAmount(), tezos.Zero)
}
