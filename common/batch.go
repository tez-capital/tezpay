package common

import (
	"errors"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

type batchBlueprint struct {
	Payouts        []PayoutRecipe
	UsedStorage    int64
	UsedGas        int64
	TransactionFee int64
	Op             *codec.Op
	limits         OperationLimits
}

func newBatch(limits *OperationLimits) batchBlueprint {
	return batchBlueprint{
		Payouts:     make([]PayoutRecipe, 0),
		UsedStorage: 0,
		UsedGas:     0,
		Op:          codec.NewOp().WithSource(tezos.ZeroAddress), // dummy address
		limits: OperationLimits{
			HardGasLimitPerOperation:     limits.HardGasLimitPerOperation * 95 / 100,     // little reserve
			HardStorageLimitPerOperation: limits.HardStorageLimitPerOperation * 95 / 100, // little reserve
			MaxOperationDataLength:       limits.MaxOperationDataLength * 95 / 100,       // little reserve
		},
	}
}

func (b *batchBlueprint) AddPayout(payout PayoutRecipe) bool {
	if b.UsedStorage+payout.OpLimits.StorageLimit >= b.limits.HardStorageLimitPerOperation {
		return false
	}
	if b.UsedGas+payout.OpLimits.GasLimit >= b.limits.HardGasLimitPerOperation {
		return false
	}
	InjectTransferContents(b.Op, payout.Recipient, &payout)
	if len(b.Op.Bytes()) > b.limits.MaxOperationDataLength {
		return false
	}
	b.UsedStorage += payout.OpLimits.StorageLimit
	b.UsedGas += payout.OpLimits.GasLimit
	b.TransactionFee += payout.OpLimits.TransactionFee
	b.Payouts = append(b.Payouts, payout)
	return true
}

func (b *batchBlueprint) ToBatch() RecipeBatch {
	return b.Payouts
}

type RecipeBatch []PayoutRecipe

func SplitIntoBatches(payouts []PayoutRecipe, limits *OperationLimits) ([]RecipeBatch, error) {
	batches := make([]RecipeBatch, 0)
	batchBlueprint := newBatch(limits)

	for _, payout := range payouts {
		if !batchBlueprint.AddPayout(payout) {
			batches = append(batches, batchBlueprint.ToBatch())
			batchBlueprint = newBatch(limits)
			if !batchBlueprint.AddPayout(payout) {
				return nil, errors.New("payout did not fit the batch, this should never happen")
			}
		}
	}
	// append last
	batches = append(batches, batchBlueprint.ToBatch())

	return lo.Filter(batches, func(batch RecipeBatch, _ int) bool {
		return len(batch) > 0
	}), nil
}

func (b *RecipeBatch) ToOpExecutionContext(signer SignerEngine, transactor TransactorEngine) (*OpExecutionContext, error) {
	op := codec.NewOp().WithSource(signer.GetPKH())
	op.WithTTL(constants.MAX_OPERATION_TTL)
	for _, p := range *b {
		InjectTransferContents(op, p.Recipient, &p)
	}
	op.WithLimits(lo.Map(*b, func(p PayoutRecipe, i int) tezos.Limits {
		buffer := int64(-constants.TX_DESERIALIZATION_GAS_BUFFER) // include in first substract from rest
		if i == 0 {
			buffer = int64(constants.TX_DESERIALIZATION_GAS_BUFFER * len(*b))
		}
		return tezos.Limits{
			Fee:          p.OpLimits.TransactionFee,
			GasLimit:     p.OpLimits.GasLimit + buffer,
			StorageLimit: p.OpLimits.StorageLimit,
		}
	}), 0)

	err := transactor.Complete(op, signer.GetKey())
	if err != nil {
		return nil, err
	}
	log.Tracef("op: %x", op.Bytes())
	err = signer.Sign(op)
	if err != nil {
		return nil, err
	}
	return InitOpExecutionContext(op, transactor), nil
}
