package constants

const (
	TEZPAY_REPOSITORY = "tez-capital/tezpay"

	MUTEZ_FACTOR = 1000000

	DELEGATION_CAPACITY_FACTOR = 10

	DEFAULT_BAKER_FEE                     = float64(.05)
	DEFAULT_DELEGATOR_MINIMUM_BALANCE     = float64(0)
	DEFAULT_PAYOUT_MINIMUM_AMOUNT         = float64(0)
	DEFAULT_RPC_URL                       = "https://eu.rpc.tez.capital/"
	DEFAULT_TZKT_URL                      = "https://api.tzkt.io/"
	DEFAULT_EXPLORER_URL                  = "https://tzkt.io/"
	DEFAULT_REQUIRED_CONFIRMATIONS        = int64(2)
	DEFAULT_TX_GAS_LIMIT_BUFFER           = 98
	DEFAULT_TX_DESERIALIZATION_GAS_BUFFER = 2 // just because of integer division
	DEFAULT_TX_FEE_BUFFER                 = 10
	DEFAULT_KT_TX_FEE_BUFFER              = 50

	PAYOUT_FEE_BUFFER  = 1000 // buffer per payout to check baker balance is sufficient
	MAX_OPERATION_TTL  = 12   // 12 blocks
	ALLOCATION_STORAGE = 257

	DEFAULT_CYCLE_MONITOR_MAXIMUM_DELAY = int64(1500)
	DEFAULT_CYCLE_MONITOR_MINIMUM_DELAY = int64(500)

	CONFIG_FILE_BACKUP_SUFFIX = ".backup"
	PAYOUT_REPORT_FILE_NAME   = "payouts.csv"
	INVALID_REPORT_FILE_NAME  = "invalid.csv"
	REPORT_SUMMARY_FILE_NAME  = "summary.json"
	REPORTS_DIRECTORY         = "reports"

	DEFAULT_DONATION_ADDRESS    = "tz1UGkfyrT9yBt6U5PV7Qeui3pt3a8jffoWv"
	DEFAULT_DONATION_PERCENTAGE = 0.05
)
