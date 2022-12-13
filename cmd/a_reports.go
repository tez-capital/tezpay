package cmd

import (
	"os"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/reports"
	"github.com/alis-is/tezpay/utils"
)

func loadPastPayoutReports(baker tezos.Address, cycle int64) ([]common.PayoutReport, error) {
	reports, err := reports.ReadPayoutReports(cycle)
	if err == nil || os.IsNotExist(err) {
		return utils.FilterReportsByBaker(reports, baker), nil
	}
	return []common.PayoutReport{}, err
}
