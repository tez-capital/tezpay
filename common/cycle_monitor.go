package common

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/tez-capital/tezpay/constants"
	"github.com/trilitech/tzgo/rpc"
)

type CycleMonitorOptions struct {
	NotificationDelay int64
	CheckFrequency    int64
}

type cycleMonitor struct {
	Cycle         chan int64
	ctx           context.Context
	cancelContext context.CancelFunc
	rpc           *rpc.Client
	options       CycleMonitorOptions
}

func NewCycleMonitor(ctx context.Context, rpc *rpc.Client, options CycleMonitorOptions) (CycleMonitor, error) {
	if options.NotificationDelay == 0 {
		options.NotificationDelay = rand.Int63n(constants.DEFAULT_CYCLE_MONITOR_MAXIMUM_DELAY-constants.DEFAULT_CYCLE_MONITOR_MINIMUM_DELAY) + constants.DEFAULT_CYCLE_MONITOR_MINIMUM_DELAY
	}

	if options.CheckFrequency < 2 {
		options.CheckFrequency = 2
	}
	if options.CheckFrequency > 120 {
		options.CheckFrequency = 120
	}

	ctx, cancel := context.WithCancel(ctx)
	slog.Info("Initialized cycle monitor", "delay", options.NotificationDelay, "check_frequency", options.CheckFrequency)
	monitor := &cycleMonitor{
		Cycle:         make(chan int64),
		ctx:           ctx,
		rpc:           rpc,
		cancelContext: cancel,
		options:       options,
	}
	return monitor, monitor.CreateBlockHeaderMonitor()
}

func (monitor *cycleMonitor) Cancel() {
	slog.Debug("cycle monitoring canceled")
	monitor.Terminate()
}
func (monitor *cycleMonitor) Terminate() {
	monitor.cancelContext()
	close(monitor.Cycle)
}

func (monitor *cycleMonitor) CreateBlockHeaderMonitor() error {
	ctx := monitor.ctx

	go func() {
		// var lastProcessedCycle int64

		for ctx.Err() == nil {
			metadata, err := monitor.rpc.GetBlockMetadata(ctx, rpc.Head)
			if err != nil {
				slog.Error("failed to fetch head metadata", "error", err)
				time.Sleep(time.Second * 10)
				continue
			}
			cycle := metadata.LevelInfo.Cycle

			if metadata.LevelInfo.CyclePosition >= monitor.options.NotificationDelay {
				monitor.Cycle <- cycle
			} else {
				monitor.Cycle <- cycle - 1
			}

			slog.Debug("received new head", "level", metadata.LevelInfo.Level)
			select {
			case <-ctx.Done():
			case <-time.After(time.Second * time.Duration(monitor.options.CheckFrequency) * 30):
			}
		}
	}()
	return nil
}

func waitForNextCompletedCycle(lastProcessedCompletedCycle int64, monitor CycleMonitor) (int64, bool) {
	currentCycle := int64(0)
	var ok bool
	for lastProcessedCompletedCycle >= currentCycle-1 {
		if currentCycle > 0 {
			slog.Debug("cycle monitor - update", "current_cycle", currentCycle, "last_processed", lastProcessedCompletedCycle)
		}
		currentCycle, ok = <-monitor.GetCycleChannel()
		if !ok {
			return -1, ok
		}
	}
	return currentCycle - 1, ok
}

func (monitor *cycleMonitor) WaitForNextCompletedCycle(lastProcessedCycle int64) (int64, error) {
	cycle, ok := waitForNextCompletedCycle(lastProcessedCycle, monitor)
	if !ok {
		return -1, constants.ErrMonitoringCanceled
	}
	return cycle, nil
}

func (monitor *cycleMonitor) GetCycleChannel() chan int64 {
	return monitor.Cycle
}
