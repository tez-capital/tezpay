package common

import (
	"context"
	"math/rand"
	"os"
	"time"

	"blockwatch.cc/tzgo/rpc"
	log "github.com/sirupsen/logrus"
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
		rand.Seed(time.Now().UnixNano())
		options.NotificationDelay = int64(rand.Intn(120) + 5)
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
	return monitor, monitor.CreateBlockHeaderMonitor()
}

func (monitor *cycleMonitor) Cancel() {
	log.Warn("cycle monitoring canceled")
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

func waitForNextCompletedCycle(lastProcessedCompletedCycle int64, monitor CycleMonitor) (int64, bool) {
	currentCycle := int64(0)
	var ok bool
	for lastProcessedCompletedCycle >= currentCycle-1 {
		if currentCycle > 0 {
			log.Debugf("current cycle %d, last processed %d", currentCycle, lastProcessedCompletedCycle)
		}
		currentCycle, ok = <-monitor.GetCycleChannel()
		if !ok {
			return -1, false
		}
	}
	return currentCycle - 1, true
}

func (monitor *cycleMonitor) WaitForNextCompletedCycle(lastProcessedCycle int64) int64 {
	cycle, ok := waitForNextCompletedCycle(lastProcessedCycle, monitor)
	if !ok {
		os.Exit(0)
	}
	return cycle
}

func (monitor *cycleMonitor) GetCycleChannel() chan int64 {
	return monitor.Cycle
}
