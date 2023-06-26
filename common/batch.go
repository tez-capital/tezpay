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
	Payouts     []PayoutRecipe
	UsedStorage int64
	UsedGas     int64
	Op          *codec.Op
	limits      OperationLimits
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
	if b.UsedGas+payout.OpLimits.GasLimit+payout.OpLimits.SerializationFee >= b.limits.HardGasLimitPerOperation {
		return false
	}
	InjectTransferContents(b.Op, payout.Recipient, &payout)
	if len(b.Op.Bytes()) > b.limits.MaxOperationDataLength {
		return false
	}
	b.UsedStorage += payout.OpLimits.StorageLimit
	b.UsedGas += payout.OpLimits.GasLimit + payout.OpLimits.SerializationFee
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
	// shuffle payouts - first payout pays for signature and deserialization so we want to spread randomly (for statistics tools, they may missreport first tx like overpaid)
	payouts := lo.Shuffle(*b)
	for _, p := range payouts {
		InjectTransferContents(op, p.Recipient, &p)
	}

	serializationFee := lo.Reduce(*b, func(acc int64, p PayoutRecipe, _ int) int64 {
		return acc + p.OpLimits.SerializationFee
	}, int64(0))

	op.WithLimits(lo.Map(payouts, func(p PayoutRecipe, i int) tezos.Limits {
		buffer := int64(0)
		if i == 0 {
			buffer = serializationFee
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
