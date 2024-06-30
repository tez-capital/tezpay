package utils

import (
	"encoding/json"
	"log/slog"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

type JsonPayouts struct {
	Invalid []common.PayoutRecipe `json:"invalid,omitempty"`
	Valid   []common.PayoutRecipe `json:"valid,omitempty"`
}

type PayoutConstraint interface {
	common.PayoutRecipe | common.PayoutReport
}

func PayoutBlueprintToJson(payoutBlueprint *common.CyclePayoutBlueprint) []byte {
	marshaled, _ := json.Marshal(payoutBlueprint)
	return marshaled
}

func PayoutBlueprintFromJson(data []byte) (*common.CyclePayoutBlueprint, error) {
	var payuts common.CyclePayoutBlueprint
	err := json.Unmarshal(data, &payuts)
	if err != nil {
		return nil, err
	}
	return &payuts, err
}

func PayoutsFromJson(data []byte) ([]common.PayoutRecipe, error) {
	var payuts []common.PayoutRecipe
	err := json.Unmarshal(data, &payuts)
	if err != nil {
		return []common.PayoutRecipe{}, err
	}
	return payuts, err
}

func FilterPayoutsByTxKind(payouts []common.PayoutRecipe, kinds []enums.EPayoutTransactionKind) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return lo.Contains(kinds, payout.TxKind)
	})
}

func RejectPayoutsByTxKind(payouts []common.PayoutRecipe, kinds []enums.EPayoutTransactionKind) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return !lo.Contains(kinds, payout.TxKind)
	})
}

func FilterPayoutsByKind(payouts []common.PayoutRecipe, kinds []enums.EPayoutKind) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return lo.Contains(kinds, payout.Kind)
	})
}

func RejectPayoutsByKind(payouts []common.PayoutRecipe, kinds []enums.EPayoutKind) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return !lo.Contains(kinds, payout.Kind)
	})
}

func FilterPayoutsByType(payouts []common.PayoutRecipe, t tezos.AddressType) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return payout.Recipient.Type() == t
	})
}

func RejectPayoutsByType(payouts []common.PayoutRecipe, t tezos.AddressType) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return payout.Recipient.Type() != t
	})
}

func FilterPayoutsByCycle(payouts []common.PayoutRecipe, cycle int64) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return payout.Cycle == cycle
	})
}

func OnlyValidPayouts(payouts []common.PayoutRecipe) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return payout.IsValid
	})
}

func OnlyInvalidPayouts(payouts []common.PayoutRecipe) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return !payout.IsValid
	})
}

func FilterReportsByBaker(payouts []common.PayoutReport, t tezos.Address) []common.PayoutReport {
	return lo.Filter(payouts, func(payout common.PayoutReport, _ int) bool {
		return payout.Baker.Equal(t)
	})
}

func FilterReportsByCycle(payouts []common.PayoutReport, cycle int64) []common.PayoutReport {
	return lo.Filter(payouts, func(payout common.PayoutReport, _ int) bool {
		return payout.Cycle == cycle
	})
}

type payoutId struct {
	kind     enums.EPayoutKind
	txKind   enums.EPayoutTransactionKind
	contract string
	token    string
	address  string
}

func FilterRecipesByReports(payouts []common.PayoutRecipe, reports []common.PayoutReport, collector common.CollectorEngine) ([]common.PayoutRecipe, []common.PayoutReport) {
	paidOut := make(map[payoutId]common.PayoutReport)
	validOpHashes := make(map[string]bool)
	if collector == nil {
		slog.Debug("collector undefined filtering payout recipes only by succcess status from reports")
	}

	for _, report := range reports {
		addr := report.Delegator.String()
		if report.Delegator.Equal(tezos.ZeroAddress) {
			addr = report.Recipient.String()
		}
		payoutId := payoutId{report.Kind, report.TxKind, report.FAContract.String(), report.FATokenId.String(), addr}
		if collector != nil && !report.OpHash.Equal(tezos.ZeroOpHash) {
			if _, ok := validOpHashes[report.OpHash.String()]; ok {
				paidOut[payoutId] = report
				continue
			}

			slog.Debug("checking with collector whether operation applied", "collector", collector.GetId(), "op_hash", report.OpHash.String())
			paid, err := collector.WasOperationApplied(report.OpHash)
			if err != nil {
				slog.Warn("collector check failed", "op_hash", report.OpHash.String(), "error", err.Error())
			}
			if paid == common.OPERATION_STATUS_APPLIED {
				paidOut[payoutId] = report
				validOpHashes[report.OpHash.String()] = true
			}
			// NOTE: in case we would like to rely only on collector status we could continue here
			// but reports are fairly reliable so we will continue to check them rn
			// continue
		}

		if report.IsSuccess {
			paidOut[payoutId] = report
			validOpHashes[report.OpHash.String()] = true
		}
	}

	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		addr := payout.Delegator.String()
		if payout.Delegator.Equal(tezos.ZeroAddress) {
			addr = payout.Recipient.String()
		}
		payoutId := payoutId{payout.Kind, payout.TxKind, payout.FAContract.String(), payout.FATokenId.String(), addr}
		_, ok := paidOut[payoutId]
		return !ok
	}), lo.Values(paidOut)
}
