package common

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tez-capital/tezpay/test/goroutineleak"
)

func TestMain(m *testing.M) {
	code := m.Run()
	if code == 0 {
		if err := goroutineleak.Check(time.Second); err != nil {
			fmt.Fprintln(os.Stderr, err)
			code = 1
		}
	}
	os.Exit(code)
}
