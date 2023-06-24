package utils

import (
	"net/url"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/samber/lo"
)

func EstimateContentFee(content codec.Operation, costs tezos.Costs, params *tezos.Params, serializationGas int64, withTxBuffer bool) int64 {
	// we add deserialization buffer to gas limit because it is substracted for all tx before broadcast and added to the first tx limit
	total := codec.CalculateMinFee(content, costs.GasUsed+constants.TX_GAS_LIMIT_BUFFER+serializationGas, true, params)
	if withTxBuffer {
		return total + constants.TRANSACTION_FEE_BUFFER
	}
	return total
}

func EstimateTransactionFee(op *codec.Op, costs []tezos.Costs, serializationGas int64) int64 {
	gasFee := lo.Reduce(op.Contents, func(agg int64, content codec.Operation, i int) int64 {
		return agg + EstimateContentFee(content, costs[i], op.Params, serializationGas, false)
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

func GetOpReference(opHash tezos.OpHash, explorer string) string {
	reference := opHash.String()
	if explorer != "" {
		reference, _ = url.JoinPath(explorer, opHash.String())
	}
	return reference
}
