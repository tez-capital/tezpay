package utils

import (
	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/samber/lo"
)

func EstimateTransactionFee(op *codec.Op, costs []tezos.Costs) int64 {
	gasFee := lo.Reduce(op.Contents, func(agg int64, content codec.Operation, i int) int64 {
		return agg + codec.CalculateMinFee(content, costs[i].GasUsed+constants.GAS_LIMIT_BUFFER, true, op.Params)
	}, 0)
	return gasFee + constants.TRANSACTION_FEE_BUFFER /*0mutez*/
}

func CalculateStorageLimit(costs tezos.Costs) int64 {
	limit := costs.StorageUsed
	if costs.AllocationBurn > 0 {
		limit += constants.ALLOCATION_STORAGE
	}
	return limit
}
