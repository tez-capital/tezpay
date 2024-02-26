package generate

import (
	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/samber/lo"
)

func splitIntoBatches[T interface{}](candidates []T, capacity int) [][]T {
	batches := make([][]T, 0)
	if capacity == 0 {
		capacity = constants.DEFAULT_SIMULATION_TX_BATCH_SIZE
	}
	for offset := 0; offset < len(candidates); offset += capacity {
		batches = append(batches, lo.Slice(candidates, offset, offset+capacity))
	}

	return batches
}

func buildOpForEstimation[T common.TransferArgs](ctx *PayoutGenerationContext, batch []T, injectBurnTransactions bool) (*codec.Op, error) {
	var err error
	op := codec.NewOp().WithSource(ctx.PayoutKey.Address())
	op.WithTTL(constants.MAX_OPERATION_TTL)
	if injectBurnTransactions {
		op.WithTransfer(tezos.BurnAddress, 1)
	}
	for _, p := range batch {
		if err = common.InjectTransferContents(op, ctx.PayoutKey.Address(), p); err != nil {
			break
		}
	}
	if injectBurnTransactions {
		op.WithTransfer(tezos.BurnAddress, 1)
	}
	return op, err
}
