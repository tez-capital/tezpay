package utils

import (
	"net/url"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/samber/lo"
)

// all buffers and additional costs should be added through txExtra
func EstimateContentFee(content codec.Operation, costs tezos.Costs, params *tezos.Params, txExtra int64) int64 {
	// we add deserialization buffer to gas limit because it is substracted for all tx before broadcast and added to the first tx limit
	return codec.CalculateMinFee(content, costs.GasUsed, true, params)
}

// all buffers and additional costs should be added through txExtra
func EstimateTransactionFee(op *codec.Op, costs []tezos.Costs, txExtra int64) int64 {
	gasFee := lo.Reduce(op.Contents, func(agg int64, content codec.Operation, i int) int64 {
		return agg + EstimateContentFee(content, costs[i], op.Params, txExtra)
	}, 0)
	return gasFee + constants.OPERATION_FEE_BUFFER /*0mutez rn*/
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
