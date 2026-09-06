package goroutineleak

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/pprof"
	"time"
)

func Check(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		runtime.GC()
		profile := pprof.Lookup("goroutineleak")
		if profile.Count() == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			var report bytes.Buffer
			if err := profile.WriteTo(&report, 2); err != nil {
				return err
			}
			return fmt.Errorf("detected leaked goroutines:\n%s", report.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
