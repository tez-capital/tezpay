package cmd

const (
	// ops
	EXIT_OPERTION_FAILED   = 1
	EXIT_OPERTION_CANCELED = 2
	EXIT_INVALID_ARGS      = 3
	EXIT_INVALID_LOG_LEVEL = 4

	// payouts io
	EXIT_PAYOUT_WRITE_FAILURE                = 10
	EXIT_PAYOUTS_READ_FAILURE                = 11
	EXIT_PAYOUT_REPORTS_PARSING_FAULURE      = 12
	EXIT_CYCLE_PAYOUT_REPORT_MARSHAL_FAILURE = 11

	// configuration
	EXIT_CONFIGURATION_LOAD_FAILURE     = 20
	EXIT_CONFIGURATION_GENERATE_FAILURE = 21
	EXIT_CONFIGURATION_SAVE_FAILURE     = 22

	EXIT_STATE_LOAD_FAILURE = 30
)
