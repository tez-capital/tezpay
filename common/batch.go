package common

import (
	"errors"
	"log/slog"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/constants"
	"github.com/trilitech/tzgo/codec"
	"github.com/trilitech/tzgo/tezos"
)

type batchBlueprint struct {
	Payouts     []*AccumulatedPayoutRecipe
	UsedStorage int64
	UsedGas     int64
	Op          *codec.Op
	limits      OperationLimits
}

func NewBatch(limits *OperationLimits, metadataDeserializationGasLimit int64) batchBlueprint {
	return batchBlueprint{
		Payouts:     make([]*AccumulatedPayoutRecipe, 0),
		UsedStorage: 0,
		UsedGas:     metadataDeserializationGasLimit,
		Op:          codec.NewOp().WithSource(tezos.ZeroAddress).WithBranch(tezos.MustParseBlockHash("BM4VEjb3EGdgNgJhwfVUsUqPYvZWJUHdmKKgabuDkwy6SmUKDve")), // dummy address
		limits: OperationLimits{
			HardGasLimitPerOperation:     limits.HardGasLimitPerOperation * 95 / 100,     // little reserve
			HardStorageLimitPerOperation: limits.HardStorageLimitPerOperation * 95 / 100, // little reserve
			MaxOperationDataLength:       limits.MaxOperationDataLength * 95 / 100,       // little reserve
		},
	}
}

func (b *batchBlueprint) AddPayout(payout *AccumulatedPayoutRecipe) bool {
	if b.UsedStorage+payout.OpLimits.StorageLimit >= b.limits.HardStorageLimitPerOperation {
		return false
	}
	if b.UsedGas+payout.OpLimits.GasLimit+payout.OpLimits.DeserializationGasLimit >= b.limits.HardGasLimitPerOperation {
		return false
	}
	InjectTransferContents(b.Op, payout.Recipient, payout)
	if len(b.Op.Bytes()) > b.limits.MaxOperationDataLength-constants.DEFAULT_BATCHING_OPERATION_DATA_BUFFER {
		return false
	}
	b.UsedStorage += payout.OpLimits.StorageLimit
	b.UsedGas += payout.OpLimits.GasLimit + payout.OpLimits.DeserializationGasLimit
	b.Payouts = append(b.Payouts, payout)

	return true
}

func (b *batchBlueprint) ToBatch() RecipeBatch {
	return b.Payouts
}

type RecipeBatch []*AccumulatedPayoutRecipe

func (b *RecipeBatch) ToOpExecutionContext(signer SignerEngine, transactor TransactorEngine) (*OpExecutionContext, error) {
	op := codec.NewOp().WithSource(signer.GetPKH())
	op.WithTTL(constants.MAX_OPERATION_TTL)

	serializationGasLimit := lo.Reduce(*b, func(acc int64, p *AccumulatedPayoutRecipe, _ int) int64 {
		return acc + p.OpLimits.DeserializationGasLimit
	}, int64(0))

	for i, p := range *b {
		buffer := int64(0)
		if i == 0 {
			buffer = serializationGasLimit
		}
		InjectTransferContentsWithLimits(op, signer.GetPKH(), p, tezos.Limits{
			Fee:          p.OpLimits.TransactionFee,
			GasLimit:     p.OpLimits.GasLimit + buffer,
			StorageLimit: p.OpLimits.StorageLimit,
		})
	}

	err := transactor.Complete(op, signer.GetKey())
	if err != nil {
		return nil, errors.Join(constants.ErrFailedToCompleteOperation, err)
	}

	slog.Debug("new op context", "op", op.Bytes(), "op_hash", op.Hash())
	err = signer.Sign(op)
	if err != nil {
		return nil, errors.Join(constants.ErrFailedToSignOperation, err)
	}
	return InitOpExecutionContext(op, transactor), nil
}
