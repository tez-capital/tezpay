package generate

import (
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/trilitech/tzgo/codec"
	"github.com/trilitech/tzgo/tezos"
)

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
