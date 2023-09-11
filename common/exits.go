package common

import (
	"encoding/json"
	"fmt"
)

type PanicStatus struct {
	ExitCode int
	Error    error
	Message  string
}

const (
	EXIT_SUCCESS        = 0
	EXIT_COMMON_FAILURE = 1
	EXIT_IVNALID_ARGS   = 2
	// ops
	EXIT_OPERTION_FAILED   = 5
	EXIT_OPERTION_CANCELED = 6

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

	EXIT_UNHANDLED_ERROR = 100
)

func RaceConditionPanicWithMetadata(reason string, id string, metadata ...interface{}) {
	fmt.Printf("%s - metadata %s:\n", reason, id)
	for _, m := range metadata {
		data, err := json.Marshal(m)
		if err == nil {
			fmt.Println(string(data))
			continue
		}
		fmt.Printf("Failed to marshal metadata: %s\n", err)
	}
	fmt.Printf("Please report above metadata to the developers.\n")
	panic(PanicStatus{
		ExitCode: EXIT_UNHANDLED_ERROR,
		Error:    fmt.Errorf("%s - metadata %s", reason, id),
	})
}
