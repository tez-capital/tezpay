package ops

import (
	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/clients/interfaces"
	tezpay_tezos "github.com/alis-is/tezpay/clients/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core/payout/common"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

type batchBlueprint struct {
	Payouts        []common.PayoutRecipe
	UsedStorage    int64
	UsedGas        int64
	TransactionFee int64
	Op             *codec.Op
	limits         tezpay_tezos.OperationLimits
}

func newBatch(limits *tezpay_tezos.OperationLimits) batchBlueprint {
	return batchBlueprint{
		Payouts:     make([]common.PayoutRecipe, 0),
		UsedStorage: 0,
		UsedGas:     0,
		Op:          codec.NewOp().WithSource(tezos.ZeroAddress), // dummy address
		limits: tezpay_tezos.OperationLimits{
			HardGasLimitPerOperation:     limits.HardGasLimitPerOperation * 95 / 100,     // little reserve
			HardStorageLimitPerOperation: limits.HardStorageLimitPerOperation * 95 / 100, // little reserve
			MaxOperationDataLength:       limits.MaxOperationDataLength * 95 / 100,       // little reserve
		},
	}
}

func (b *batchBlueprint) AddPayout(payout common.PayoutRecipe) bool {
	if b.UsedStorage+payout.OpLimits.StorageLimit >= b.limits.HardStorageLimitPerOperation {
		return false
	}
	if b.UsedGas+payout.OpLimits.GasLimit >= b.limits.HardGasLimitPerOperation {
		return false
	}
	b.Op.WithTransfer(payout.Recipient, payout.Amount.Int64())
	if len(b.Op.Bytes()) > b.limits.MaxOperationDataLength {
		return false
	}
	b.UsedStorage += payout.OpLimits.StorageLimit
	b.UsedGas += payout.OpLimits.GasLimit
	b.TransactionFee += payout.OpLimits.TransactionFee
	b.Payouts = append(b.Payouts, payout)
	return true
}

func (b *batchBlueprint) ToBatch() Batch {
	return b.Payouts
}

type Batch []common.PayoutRecipe

func SplitIntoBatches(payouts []common.PayoutRecipe, limits *tezpay_tezos.OperationLimits) []Batch {
	batches := make([]Batch, 0)
	batchBlueprint := newBatch(limits)

	for _, payout := range payouts {
		if !batchBlueprint.AddPayout(payout) {
			batches = append(batches, batchBlueprint.ToBatch())
			batchBlueprint = newBatch(limits)
			if !batchBlueprint.AddPayout(payout) {
				// TODO: handle properly
				panic("payout did not fit the batch, this should never happen")
			}
		}
	}
	// append last
	batches = append(batches, batchBlueprint.ToBatch())

	return lo.Filter(batches, func(batch Batch, _ int) bool {
		return len(batch) > 0
	})
}

func (b *Batch) ToOpExecutionContext(signer interfaces.SignerEngine, transactor interfaces.TransactorEngine) (*OpExecutionContext, error) {
	op := codec.NewOp().WithSource(signer.GetPKH())
	for _, p := range *b {
		op.WithTransfer(p.Recipient, p.Amount.Int64())
	}
	op.WithTTL(constants.MAX_OPERATION_TTL)
	op.WithLimits(lo.Map(*b, func(p common.PayoutRecipe, _ int) tezos.Limits {
		return tezos.Limits{
			Fee:          p.OpLimits.TransactionFee,
			GasLimit:     p.OpLimits.GasLimit,
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
