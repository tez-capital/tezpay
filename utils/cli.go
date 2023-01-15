package utils

import (
	"fmt"
	"os"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/samber/lo"
)

const (
	TOTAL = "Total"
)

func shortenAddress(taddr tezos.Address) string {
	if taddr.Equal(tezos.ZeroAddress) || taddr.Equal(tezos.InvalidAddress) {
		return "-"
	}
	addr := taddr.String()
	total := len(addr)
	if total <= 13 {
		return addr
	}
	return fmt.Sprintf("%s...%s", addr[:5], addr[total-5:])
}

func toPercentage[T FloatConstraint](portion T) string {
	if portion == 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f %%", portion*100)
}

func printPayouts(payouts []common.PayoutRecipe, header string, printTotals bool) {
	if len(payouts) == 0 {
		return
	}
	payoutTable := table.NewWriter()
	payoutTable.SetStyle(table.StyleLight)
	payoutTable.SetColumnConfigs([]table.ColumnConfig{{Number: 1, Align: text.AlignLeft}, {Number: 2, Align: text.AlignLeft}})
	payoutTable.SetOutputMirror(os.Stdout)
	payoutTable.SetTitle(header)
	payoutTable.Style().Title.Align = text.AlignCenter
	payoutTable.AppendHeader(table.Row{"Delegator", "Recipient", "Delegated Balance", "Kind", "Amount", "Fee Rate", "Baker Fee", "Transaction Fee", "Note"}, table.RowConfig{AutoMerge: true})
	for _, payout := range payouts {
		note := payout.Note
		if note == "" {
			note = "-"
		}
		txFee := int64(0)
		if payout.OpLimits != nil {
			txFee = payout.OpLimits.TransactionFee
		}

		payoutTable.AppendRow(table.Row{shortenAddress(payout.Delegator), shortenAddress(payout.Recipient), MutezToTezS(payout.DelegatedBalance.Int64()), payout.Kind, MutezToTezS(payout.Amount.Int64()), toPercentage(payout.FeeRate), MutezToTezS(payout.Fee.Int64()), MutezToTezS(txFee), note}, table.RowConfig{AutoMerge: false})
	}
	if printTotals {
		payoutTable.AppendSeparator()
		totalAmount := lo.Reduce(payouts, func(agg int64, payout common.PayoutRecipe, _ int) int64 {
			return agg + payout.Amount.Int64()
		}, 0)
		bakerFee := lo.Reduce(payouts, func(agg int64, payout common.PayoutRecipe, _ int) int64 {
			return agg + payout.Fee.Int64()
		}, 0)
		transactionFees := lo.Reduce(payouts, func(agg int64, payout common.PayoutRecipe, _ int) int64 {
			return agg + payout.OpLimits.TransactionFee
		}, 0)
		payoutTable.AppendRow(table.Row{TOTAL, TOTAL, TOTAL, TOTAL, MutezToTezS(totalAmount), "-", MutezToTezS(bakerFee), MutezToTezS(transactionFees), "-"}, table.RowConfig{AutoMerge: true})
	}
	payoutTable.Render()
}

// print invalid payouts
func PrintInvalidPayoutRecipes(payouts []common.PayoutRecipe, cycle int64) {
	printPayouts(OnlyInvalidPayouts(payouts), fmt.Sprintf("Invalid - #%d", cycle), false)
}

// print payable payouts
func PrintValidPayoutRecipes(payouts []common.PayoutRecipe, cycle int64) {
	printPayouts(OnlyValidPayouts(payouts), fmt.Sprintf("Valid - #%d", cycle), true)
}

func PrintPayoutsAsJson[T PayoutConstraint](payouts []T) {
	fmt.Println(string(PayoutsToJson(payouts)))
}

func IsTty() bool {
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		return true
	} else {
		return false
	}
}

func PrintReports(payouts []common.PayoutReport, header string, printTotals bool) {
	if len(payouts) == 0 {
		return
	}
	payoutTable := table.NewWriter()
	payoutTable.SetStyle(table.StyleLight)
	payoutTable.SetColumnConfigs([]table.ColumnConfig{{Number: 1, Align: text.AlignLeft}, {Number: 2, Align: text.AlignLeft}})
	payoutTable.SetOutputMirror(os.Stdout)
	payoutTable.SetTitle(header)
	payoutTable.Style().Title.Align = text.AlignCenter
	payoutTable.AppendHeader(table.Row{"Delegator", "Recipient", "Kind", "Amount", "Baker Fee", "Transaction Fee", "OpHash"}, table.RowConfig{AutoMerge: true})
	for _, payout := range payouts {
		note := payout.Note
		if note == "" {
			note = "-"
		}
		txFee := int64(0)

		payoutTable.AppendRow(table.Row{shortenAddress(payout.Delegator), shortenAddress(payout.Recipient), payout.Kind, MutezToTezS(payout.Amount.Int64()), MutezToTezS(payout.Fee.Int64()), MutezToTezS(txFee), payout.OpHash}, table.RowConfig{AutoMerge: false})
	}
	if printTotals {
		payoutTable.AppendSeparator()
		totalAmount := lo.Reduce(payouts, func(agg int64, payout common.PayoutReport, _ int) int64 {
			return agg + payout.Amount.Int64()
		}, 0)
		bakerFee := lo.Reduce(payouts, func(agg int64, payout common.PayoutReport, _ int) int64 {
			return agg + payout.Fee.Int64()
		}, 0)
		transactionFees := lo.Reduce(payouts, func(agg int64, payout common.PayoutReport, _ int) int64 {
			return agg + payout.TransactionFee
		}, 0)
		payoutTable.AppendRow(table.Row{TOTAL, TOTAL, TOTAL, MutezToTezS(totalAmount), MutezToTezS(bakerFee), MutezToTezS(transactionFees), "-"}, table.RowConfig{AutoMerge: true})
	}
	payoutTable.Render()
}

func PrintCycleSummary(summary common.CyclePayoutSummary, header string) {
	summaryTable := table.NewWriter()
	summaryTable.SetStyle(table.StyleLight)
	summaryTable.SetColumnConfigs([]table.ColumnConfig{{Number: 1, Align: text.AlignLeft}, {Number: 2, Align: text.AlignRight}})
	summaryTable.SetOutputMirror(os.Stdout)
	summaryTable.SetTitle(header)
	summaryTable.Style().Title.Align = text.AlignCenter
	summaryTable.AppendRow(table.Row{"Earned Fees", MutezToTezS(summary.EarnedFees.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Earned Rewards", MutezToTezS(summary.EarnedRewards.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Distributed Rewards", MutezToTezS(summary.DistributedRewards.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendSeparator()
	summaryTable.AppendRow(table.Row{"Donated Bonds", MutezToTezS(summary.DonatedBonds.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Donated Fees", MutezToTezS(summary.DonatedFees.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Donated Total", MutezToTezS(summary.DonatedTotal.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendSeparator()
	summaryTable.AppendRow(table.Row{"Bond Income", MutezToTezS(summary.BondIncome.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Fee Income", MutezToTezS(summary.FeeIncome.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Income Total", MutezToTezS(summary.IncomeTotal.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.Render()
}

func PrintBatchResults(results []common.BatchResult, header string, explorerUrl string) {
	if len(results) == 0 {
		return
	}
	resultsTable := table.NewWriter()
	resultsTable.SetStyle(table.StyleLight)
	resultsTable.SetColumnConfigs([]table.ColumnConfig{{Number: 1, Align: text.AlignLeft}, {Number: 2, Align: text.AlignLeft}})
	resultsTable.SetOutputMirror(os.Stdout)
	resultsTable.SetTitle(header)
	resultsTable.Style().Title.Align = text.AlignCenter
	resultsTable.AppendHeader(table.Row{"n.", "Transactions", "Success", "Reference"}, table.RowConfig{AutoMerge: true})
	for i, result := range results {
		resultsTable.AppendRow(table.Row{i + 1, len(result.Payouts), result.IsSuccess, GetOpReference(result.OpHash, explorerUrl)}, table.RowConfig{AutoMerge: false})
	}
	resultsTable.Render()
}
