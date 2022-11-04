package utils

import (
	"encoding/json"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
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

func PayoutsToJson[T PayoutConstraint](payouts []T) []byte {
	marshaled, _ := json.Marshal(payouts)
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

func FilterPayoutsByType(payouts []common.PayoutRecipe, t tezos.AddressType) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return payout.Recipient.Type == t
	})
}

func FilterPayoutsByCycle(payouts []common.PayoutRecipe, cycle int64) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return payout.Cycle == cycle
	})
}

func RejectPayoutsByType(payouts []common.PayoutRecipe, t tezos.AddressType) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return payout.Recipient.Type != t
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

func FilterRecipesByReports(payouts []common.PayoutRecipe, reports []common.PayoutReport, collector common.CollectorEngine) ([]common.PayoutRecipe, []common.PayoutReport) {
	paidOut := make(map[string]common.PayoutReport)
	validOpHashes := make(map[string]bool)
	if collector == nil {
		log.Debugf("collector undefined filtering payout recipes only by succcess status from reports")
	}
	for _, report := range reports {
		k := report.Delegator.String()
		if collector != nil && !report.OpHash.Equal(tezos.ZeroOpHash) {
			if _, ok := validOpHashes[report.OpHash.String()]; ok {
				paidOut[k] = report
				continue
			}

			log.Debugf("checking with '%s' whether operation '%s' applied", collector.GetId(), report.OpHash)
			paid, err := collector.WasOperationApplied(report.OpHash)
			if err != nil {
				log.Warnf("collector check of '%s' failed", report.OpHash)
			}
			if paid {
				paidOut[k] = report
				validOpHashes[report.OpHash.String()] = true
			}
		}

		if report.IsSuccess {
			paidOut[k] = report
			validOpHashes[report.OpHash.String()] = true
		}
	}
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		k := payout.Delegator.String()
		_, ok := paidOut[k]
		return !ok
	}), lo.Values(paidOut)
}
