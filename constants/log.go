package constants

import "slices"

const (
	LOG_MESSAGE_PAYOUTS_GENERATED = "payouts generated"
	LOG_MESSAGE_PREPAYOUT_SUMMARY = "pre-payout summary"
	LOG_MESSAGE_PAYOUTS_EXECUTED  = "payouts executed"

	LOG_SERVER_CACHE_CAPACITY = 50

	LOG_FIELD_PAYOUTS                 = "payouts"
	LOG_FIELD_CYCLES                  = "cycles"
	LOG_FIELD_CYCLE_PAYOUT_BLUEPRINT  = "cycle_payout_blueprint"
	LOG_FIELD_SUMMARY                 = "summary"
	LOG_FIELD_REPORTS_OF_PAST_PAYOUTS = "reports_of_past_payouts"
	LOG_FIELD_ACCUMULATED_PAYOUTS     = "accumulated_payouts"
	LOG_FIELD_VALID_PAYOUTS           = "valid_payouts"
	LOG_FIELD_INVALID_PAYOUTS         = "invalid_payouts"
	LOG_FIELD_BATCHES                 = "batches"
)

var (
	LOG_TOP_LEVEL_HIDDEN_FIELDS = []string{
		"stage",
		"phase",
	}
)

func init() {
	slices.Sort(LOG_TOP_LEVEL_HIDDEN_FIELDS)
}
