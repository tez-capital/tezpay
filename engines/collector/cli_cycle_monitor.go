package collector_engines

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"blockwatch.cc/tzgo/rpc"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	log "github.com/sirupsen/logrus"
)

type CycleMonitorCycleChannelProvider interface {
	getCycleChannel() chan int64
}

type cycleMonitor struct {
	Cycle         chan int64
	ctx           context.Context
	cancelContext context.CancelFunc
	rpc           *rpc.Client
	options       common.CycleMonitorOptions
}

func NewCycleMonitor(ctx context.Context, rpc *rpc.Client, options common.CycleMonitorOptions) (common.CycleMonitor, error) {
	if options.NotificationDelay == 0 {
		options.NotificationDelay = int64(rand.Intn(constants.CYCLE_MONITOR_MAXIMUM_DELAY) + constants.CYCLE_MONITOR_DELAY_OFFSET)
	}

	if options.CheckFrequency < 2 {
		options.CheckFrequency = 2
	}
	if options.CheckFrequency > 120 {
		options.CheckFrequency = 120
	}

	ctx, cancel := context.WithCancel(ctx)
	log.Infof("Initialized cycle monitor with ~%d blocks delay", options.NotificationDelay)
	monitor := &cycleMonitor{
		Cycle:         make(chan int64),
		ctx:           ctx,
		rpc:           rpc,
		cancelContext: cancel,
		options:       options,
	}
	return monitor, monitor.createBlockHeaderMonitor()
}

func (monitor *cycleMonitor) Cancel() {
	log.Debugf("cycle monitoring canceled")
	monitor.terminate()
}
func (monitor *cycleMonitor) terminate() {
	monitor.cancelContext()
	close(monitor.Cycle)
}

func (monitor *cycleMonitor) createBlockHeaderMonitor() error {
	ctx := monitor.ctx

	go func() {
		// var lastProcessedCycle int64

		for ctx.Err() == nil {
			metadata, err := monitor.rpc.GetBlockMetadata(ctx, rpc.Head)
			if err != nil {
				log.Errorf("failed to fetch head metadata - %s", err.Error())
				time.Sleep(time.Second * 10)
				continue
			}
			cycle := metadata.LevelInfo.Cycle

			if metadata.LevelInfo.CyclePosition >= monitor.options.NotificationDelay /* && lastProcessedCycle < cycle */ {
				monitor.Cycle <- cycle
				// lastProcessedCycle = cycle
			} else {
				monitor.Cycle <- cycle - 1
			}

			log.Tracef("received new head %d", metadata.LevelInfo.Level)
			select {
			case <-ctx.Done():
			case <-time.After(time.Second * time.Duration(monitor.options.CheckFrequency) * 30):
			}
		}
	}()
	return nil
}

func waitForNextCompletedCycle(lastProcessedCompletedCycle int64, monitor CycleMonitorCycleChannelProvider) (int64, bool) {
	currentCycle := int64(0)
	var ok bool
	for lastProcessedCompletedCycle >= currentCycle-1 {
		if currentCycle > 0 {
			log.Debugf("current cycle %d, last processed %d", currentCycle, lastProcessedCompletedCycle)
		}
		currentCycle, ok = <-monitor.getCycleChannel()
		if !ok {
			return -1, ok
		}
	}
	return currentCycle - 1, ok
}

func (monitor *cycleMonitor) WaitForNextCompletedCycle(lastProcessedCycle int64) (int64, error) {
	cycle, ok := waitForNextCompletedCycle(lastProcessedCycle, monitor)
	if !ok {
		return -1, fmt.Errorf("canceled")
	}
	return cycle, nil
}

func (monitor *cycleMonitor) getCycleChannel() chan int64 {
	return monitor.Cycle
}
