package utils

import (
	"net/url"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/samber/lo"
)

// all buffers and additional costs should be added through txExtra
func EstimateContentFee(content codec.Operation, gasUsed int64, params *tezos.Params) int64 {
	// we add deserialization buffer to gas limit because it is substracted for all tx before broadcast and added to the first tx limit
	fee := codec.CalculateMinFee(content, gasUsed, true, params)
	for content.Limits().Fee != fee {
		limits := content.Limits()
		limits.Fee = fee
		content.WithLimits(limits)
		fee = codec.CalculateMinFee(content, gasUsed, true, params)
	}
	return fee
}

// all buffers and additional costs should be added through txExtra
func EstimateTransactionFee(op *codec.Op, gasUsage []int64, txExtra int64) int64 {
	fee := lo.Reduce(op.Contents, func(agg int64, content codec.Operation, i int) int64 {
		return agg + EstimateContentFee(content, gasUsage[i], op.Params)
	}, 0)
	return fee + txExtra
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
