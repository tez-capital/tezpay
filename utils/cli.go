package utils

import (
	"fmt"
	"os"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
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

func getColumnsByIndexes[T any](row []T, indexes []int) []T {
	return lo.Filter(row, func(_ T, i int) bool {
		return lo.Contains(indexes, i)
	})
}

func columnsAsInterfaces[T any](row []T) []interface{} {
	return lo.Map(row, func(c T, _ int) interface{} {
		return c
	})
}

func replaceZeroFields[T comparable](items []T, value T, stopOnNonEmpty bool) []T {
	var zero T
	for i, item := range items {
		if item == zero {
			items[i] = value
		} else if stopOnNonEmpty {
			break
		}
	}
	return items
}

func getNonEmptyIndexes[T comparable](headers []string, data [][]T) []int {
	var zero T
	return lo.Filter(lo.Range(len(headers)), func(c int, i int) bool {
		return lo.SomeBy(data, func(d []T) bool {
			return d[i] != zero
		})
	})
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

	headers := payouts[0].GetTableHeaders()
	data := lo.Map(payouts, func(p common.PayoutRecipe, _ int) []string {
		return p.ToTableRowData()
	})
	validIndexes := getNonEmptyIndexes(headers, data)

	payoutTable.AppendHeader(columnsAsInterfaces(getColumnsByIndexes(headers, validIndexes)), table.RowConfig{AutoMerge: true})

	for _, payout := range data {
		row := replaceZeroFields(payout, "-", false)
		payoutTable.AppendRow(columnsAsInterfaces(getColumnsByIndexes(row, validIndexes)), table.RowConfig{AutoMerge: false})
	}
	if printTotals {
		payoutTable.AppendSeparator()
		totals := replaceZeroFields(common.GetRecipesTotals(payouts), TOTAL, true)
		totals = replaceZeroFields(totals, "-", false)

		payoutTable.AppendRow(columnsAsInterfaces(getColumnsByIndexes(totals, validIndexes)), table.RowConfig{AutoMerge: true})
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

	headers := payouts[0].GetTableHeaders()
	data := lo.Map(payouts, func(p common.PayoutReport, _ int) []string {
		return p.ToTableRowData()
	})
	validIndexes := getNonEmptyIndexes(headers, data)

	payoutTable.AppendHeader(columnsAsInterfaces(getColumnsByIndexes(headers, validIndexes)), table.RowConfig{AutoMerge: true})
	for _, payout := range data {
		row := replaceZeroFields(payout, "-", false)
		payoutTable.AppendRow(columnsAsInterfaces(getColumnsByIndexes(row, validIndexes)), table.RowConfig{AutoMerge: false})
	}
	if printTotals {
		payoutTable.AppendSeparator()
		totals := replaceZeroFields(common.GetReportsTotals(payouts), TOTAL, true)
		totals = replaceZeroFields(totals, "-", false)
		payoutTable.AppendRow(columnsAsInterfaces(getColumnsByIndexes(totals, validIndexes)), table.RowConfig{AutoMerge: true})
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
	summaryTable.AppendRow(table.Row{"Earned Fees", common.MutezToTezS(summary.EarnedFees.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Earned Rewards", common.MutezToTezS(summary.EarnedRewards.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Distributed Rewards", common.MutezToTezS(summary.DistributedRewards.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendSeparator()
	summaryTable.AppendRow(table.Row{"Donated Bonds", common.MutezToTezS(summary.DonatedBonds.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Donated Fees", common.MutezToTezS(summary.DonatedFees.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Donated Total", common.MutezToTezS(summary.DonatedTotal.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendSeparator()
	summaryTable.AppendRow(table.Row{"Bond Income", common.MutezToTezS(summary.BondIncome.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Fee Income", common.MutezToTezS(summary.FeeIncome.Int64())}, table.RowConfig{AutoMerge: false})
	summaryTable.AppendRow(table.Row{"Income Total", common.MutezToTezS(summary.IncomeTotal.Int64())}, table.RowConfig{AutoMerge: false})
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
