package utils

import (
	"encoding/json"
	"log/slog"
	"time"

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

func PayoutBlueprintToJson(payoutBlueprint common.CyclePayoutBlueprints) []byte {
	marshaled, _ := json.Marshal(payoutBlueprint)
	return marshaled
}

func PayoutBlueprintFromJson(data []byte) (common.CyclePayoutBlueprints, error) {
	var payuts common.CyclePayoutBlueprints
	err := json.Unmarshal(data, &payuts)
	if err != nil {
		return nil, err
	}
	return payuts, err
}

func PayoutsFromJson(data []byte) ([]common.PayoutRecipe, error) {
	var payuts []common.PayoutRecipe
	err := json.Unmarshal(data, &payuts)
	if err != nil {
		return []common.PayoutRecipe{}, err
	}
	return payuts, err
}

func FilterPayoutsByTxKind(payouts []*common.AccumulatedPayoutRecipe, kinds []enums.EPayoutTransactionKind) []*common.AccumulatedPayoutRecipe {
	return lo.Filter(payouts, func(payout *common.AccumulatedPayoutRecipe, _ int) bool {
		return lo.Contains(kinds, payout.TxKind)
	})
}

func RejectPayoutsByTxKind(payouts []*common.AccumulatedPayoutRecipe, kinds []enums.EPayoutTransactionKind) []*common.AccumulatedPayoutRecipe {
	return lo.Filter(payouts, func(payout *common.AccumulatedPayoutRecipe, _ int) bool {
		return !lo.Contains(kinds, payout.TxKind)
	})
}

func FilterPayoutsByKind(payouts []*common.AccumulatedPayoutRecipe, kinds []enums.EPayoutKind) []*common.AccumulatedPayoutRecipe {
	return lo.Filter(payouts, func(payout *common.AccumulatedPayoutRecipe, _ int) bool {
		return lo.Contains(kinds, payout.Kind)
	})
}

func RejectPayoutsByKind(payouts []*common.AccumulatedPayoutRecipe, kinds []enums.EPayoutKind) []*common.AccumulatedPayoutRecipe {
	return lo.Filter(payouts, func(payout *common.AccumulatedPayoutRecipe, _ int) bool {
		return !lo.Contains(kinds, payout.Kind)
	})
}

func FilterPayoutsByType(payouts []*common.AccumulatedPayoutRecipe, t tezos.AddressType) []*common.AccumulatedPayoutRecipe {
	return lo.Filter(payouts, func(payout *common.AccumulatedPayoutRecipe, _ int) bool {
		return payout.Recipient.Type() == t
	})
}

func RejectPayoutsByType(payouts []*common.AccumulatedPayoutRecipe, t tezos.AddressType) []*common.AccumulatedPayoutRecipe {
	return lo.Filter(payouts, func(payout *common.AccumulatedPayoutRecipe, _ int) bool {
		return payout.Recipient.Type() != t
	})
}

func FilterPayoutsByCycle(payouts []common.PayoutReport, cycle int64) []common.PayoutReport {
	return lo.Filter(payouts, func(payout common.PayoutReport, _ int) bool {
		return payout.Cycle == cycle
	})
}

func OnlyValidPayoutRecipes(payouts []common.PayoutRecipe) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return payout.IsValid
	})
}

func OnlyValidAccumulatedPayouts(payouts []*common.AccumulatedPayoutRecipe) []*common.AccumulatedPayoutRecipe {
	return lo.Filter(payouts, func(payout *common.AccumulatedPayoutRecipe, _ int) bool {
		return payout.IsValid
	})
}

func OnlyFailedOrInvalidPayouts(payouts []common.PayoutReport) []common.PayoutReport {
	return lo.Filter(payouts, func(payout common.PayoutReport, _ int) bool {
		return !payout.IsSuccess
	})
}

func OnlyInvalidPayoutRecipes(payouts []common.PayoutRecipe) []common.PayoutRecipe {
	return lo.Filter(payouts, func(payout common.PayoutRecipe, _ int) bool {
		return !payout.IsValid
	})
}

func OnlyInvalidAccumulatedPayouts(payouts []*common.AccumulatedPayoutRecipe) []*common.AccumulatedPayoutRecipe {
	return lo.Filter(payouts, func(payout *common.AccumulatedPayoutRecipe, _ int) bool {
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
	paidOut := make(map[string]common.PayoutReport)
	validOpHashes := make(map[string]bool)
	if collector == nil {
		slog.Debug("collector undefined filtering payout recipes only by succcess status from reports")
	}

	for _, report := range reports {
		payoutId := report.GetDestinationIdentifier()
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
		payoutId := payout.ToPayoutReport().GetDestinationIdentifier()
		_, ok := paidOut[payoutId]
		return !ok
	}), lo.Values(paidOut)
}

func GeneratePayoutSummary(blueprints []*common.CyclePayoutBlueprint, reports []common.PayoutReport) (summary *common.PayoutSummary) {
	allReports := lo.Flatten(lo.Map(reports, func(pr common.PayoutReport, _ int) []common.PayoutReport {
		return pr.Disperse()
	}))

	summary = &common.PayoutSummary{}
	delegators := make(map[string]struct{}, len(allReports))
	paidDelegators := make(map[string]struct{}, len(allReports))

	for _, blueprint := range blueprints {
		cycle := blueprint.Cycle

		cycleReports := FilterPayoutsByCycle(allReports, cycle)
		cycleSummary := common.CyclePayoutSummary{
			OwnStakedBalance:         blueprint.OwnStakedBalance,
			OwnDelegatedBalance:      blueprint.OwnDelegatedBalance,
			ExternalStakedBalance:    blueprint.ExternalStakedBalance,
			ExternalDelegatedBalance: blueprint.ExternalDelegatedBalance,
			EarnedBlockFees:          blueprint.EarnedBlockFees,
			EarnedRewards:            blueprint.EarnedRewards,
			EarnedTotal:              blueprint.EarnedTotal,
			BondIncome:               blueprint.BondIncome,
			FeeIncome:                blueprint.FeeIncome,
			IncomeTotal:              blueprint.IncomeTotal,
			DonatedBonds:             blueprint.DonatedBonds,
			DonatedFees:              blueprint.DonatedFees,
			DonatedTotal:             blueprint.DonatedTotal,
			Timestamp:                time.Now(),
		}
		cycleDelegators := make(map[string]struct{}, len(cycleReports))
		cyclePaidDelegators := make(map[string]struct{}, len(cycleReports))

		for _, report := range cycleReports {
			cycleDelegators[report.Delegator.String()] = struct{}{}

			switch report.Kind {
			case enums.PAYOUT_KIND_DELEGATOR_REWARD:
				if report.IsSuccess {
					cycleSummary.DistributedRewards = cycleSummary.DistributedRewards.Add(report.Amount)
					cycleSummary.TxFeesPaid = cycleSummary.TxFeesPaid.Add64(report.TxFee)
					cycleSummary.TxFeesPaidForRewards = cycleSummary.TxFeesPaidForRewards.Add64(report.TxFee)
					cyclePaidDelegators[report.Delegator.String()] = struct{}{}
				} else {
					cycleSummary.NotDistributedRewards = cycleSummary.NotDistributedRewards.Add(report.Amount)
				}
			default:
				if report.IsSuccess {
					cycleSummary.TxFeesPaid = cycleSummary.TxFeesPaid.Add64(report.TxFee)
				}
			}
		}

		cycleSummary.Delegators = len(cycleDelegators)
		cycleSummary.PaidDelegators = len(cyclePaidDelegators)

		delegators = lo.Assign(delegators, cycleDelegators)
		paidDelegators = lo.Assign(paidDelegators, cyclePaidDelegators)
		summary.AddCycleSummary(cycle, &cycleSummary)
	}

	summary.Delegators = len(delegators)
	summary.PaidDelegators = len(paidDelegators)
	return summary
}

func GeneratePayoutSummaryFromPreparationResult(result *common.PreparePayoutsResult) (summary *common.PayoutSummary) {
	if len(result.ValidPayouts) > 0 {
		panic("preparation result contains valid payouts, use GeneratePayoutSummary with execution reports instead")
	}

	invalidReports := lo.Map(result.InvalidPayouts, func(pr common.PayoutRecipe, _ int) common.PayoutReport {
		r := pr.ToPayoutReport()
		r.IsSuccess = false
		return r
	})

	return GeneratePayoutSummary(result.Blueprints, append(result.ReportsOfPastSuccessfulPayouts, invalidReports...))
}
