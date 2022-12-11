package common

import (
	"context"
	"math/rand"
	"time"

	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	log "github.com/sirupsen/logrus"
)

type CycleMonitor struct {
	Cycle              chan int64
	lastProcessedCycle int64
	delay              int64
	counter            int64
	ctx                context.Context
	cancelContext      context.CancelFunc
	rpc                *rpc.Client
	rpcMonitor         *rpc.BlockHeaderMonitor
}

func NewCycleMonitor(ctx context.Context, rpc *rpc.Client, notificationDelay int64) (*CycleMonitor, error) {
	if notificationDelay == 0 {
		notificationDelay = int64(rand.Intn(2880)+20) /* up to 1 day in lima, up to 12 hours in M */ / 20 /* we check every 20 blocks - 10/5 mins */
	}

	ctx, cancel := context.WithCancel(ctx)
	monitor := &CycleMonitor{
		Cycle:         make(chan int64),
		ctx:           ctx,
		rpc:           rpc,
		cancelContext: cancel,
		delay:         notificationDelay,
	}
	return monitor, monitor.CreateBlockHeaderMonitor()
}

func (monitor *CycleMonitor) Cancel() {
	log.Warn("cycle monitoring canceled")
	monitor.Terminate()
}
func (monitor *CycleMonitor) Terminate() {
	monitor.cancelContext()
	close(monitor.Cycle)
	monitor.rpcMonitor.Close()
}

func (monitor *CycleMonitor) Delay() int64 {
	return monitor.delay
}

func fetchBlock(ctx context.Context, c *rpc.Client, blockID tezos.BlockHash) (*rpc.BlockMetadata, error) {
	b, err := c.GetBlockMetadata(ctx, blockID)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (monitor *CycleMonitor) CreateBlockHeaderMonitor() error {
	ctx := monitor.ctx
	if monitor.rpcMonitor != nil {
		select {
		case <-monitor.rpcMonitor.Closed():
		default:
			monitor.rpcMonitor.Close()
		}
	}
	mon := rpc.NewBlockHeaderMonitor()
	if err := monitor.rpc.MonitorBlockHeader(ctx, mon); err != nil {
		return err
	}
	monitor.rpcMonitor = mon
	go func() {
		receivedHeads := int64(0)
		for ctx.Err() == nil {
			select {
			case <-mon.Closed():
				return
			default:
			}
			h, err := mon.Recv(ctx)
			if err != nil {
				attempt := 1

				for err = monitor.CreateBlockHeaderMonitor(); attempt < 5 && err != nil; attempt++ {
					log.Warnf("failed to recreate block header monitor")
					time.Sleep(time.Second * 60)
				}
				if err != nil {
					log.Fatalf("failed to monitor blocks %s", err.Error())
					monitor.Terminate()
				}
				return
			}
			if receivedHeads%20 == 0 {
				blockMetadata, err := fetchBlock(ctx, monitor.rpc, h.Hash)
				if err != nil {
					log.Errorf("failed to fetch block metadata %d - %s", h.Level, err.Error())
					continue
				}
				cycle := blockMetadata.LevelInfo.Cycle
				if cycle > monitor.lastProcessedCycle {
					log.Debugf("new cycle %d (delay: %d) found", cycle, monitor.delay)
					monitor.counter = 1
				}

				if monitor.counter == monitor.delay {
					monitor.Cycle <- cycle
				}
				monitor.counter++
			}
			receivedHeads++
		}
	}()
	return nil
}
