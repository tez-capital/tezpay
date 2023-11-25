package common

import (
	"testing"

	"github.com/alis-is/tezpay/constants"
	"github.com/stretchr/testify/assert"
)

type dummyCycleMonitor struct {
	Cycle chan int64
	start int64
	end   int64
}

func NewDumyCycleMonitor(start int64, end int64) CycleMonitor {
	return &dummyCycleMonitor{
		Cycle: make(chan int64),
		start: start,
		end:   end,
	}
}

func (dummyCycleMonitor) Cancel() {
}

func (dummyCycleMonitor) Terminate() {
}

func (monitor *dummyCycleMonitor) CreateBlockHeaderMonitor() error {
	go func() {
		for i := monitor.start; i <= monitor.end; i++ {
			monitor.Cycle <- i
		}
		close(monitor.Cycle)
	}()

	return nil
}

func (monitor *dummyCycleMonitor) GetCycleChannel() chan int64 {
	return monitor.Cycle
}

func (monitor *dummyCycleMonitor) WaitForNextCompletedCycle(lastProcessedCycle int64) (int64, error) {
	cycle, ok := waitForNextCompletedCycle(lastProcessedCycle, monitor)
	if !ok {
		return 0, constants.ErrMonitoringCanceled
	}
	return cycle, nil
}

func runMonitoringTest(t *testing.T, start int64, end int64, lastProcessedCycle int64, expectedCycle int64) {
	assert := assert.New(t)
	monitor := NewDumyCycleMonitor(start, end)
	assert.Nil(monitor.CreateBlockHeaderMonitor())
	cycle, _ := monitor.WaitForNextCompletedCycle(lastProcessedCycle)
	assert.Equal(expectedCycle, cycle)
}

func runMonitoringTestExpectClosed(t *testing.T, start int64, end int64, lastProcessedCycle int64, expectedCycle int64) {
	assert := assert.New(t)
	monitor := NewDumyCycleMonitor(start, end)
	assert.Nil(monitor.CreateBlockHeaderMonitor())
	_, ok := waitForNextCompletedCycle(lastProcessedCycle, monitor)
	assert.Equal(ok, false)
}

func TestWaitForNextCycle(t *testing.T) {
	runMonitoringTest(t, 1, 10, 0, 1)
	runMonitoringTest(t, 1, 5, 2, 3)
	runMonitoringTest(t, 5, 5, 3, 4)
	runMonitoringTest(t, 5, 5, 0, 4)

	runMonitoringTestExpectClosed(t, 1, 5, 4, 5)
}
