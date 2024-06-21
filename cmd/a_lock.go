package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	"code.cloudfoundry.org/filelock"
	"github.com/tez-capital/tezpay/state"
)

func lockCycle(cycle int64, unlockStore *func() error, resultChan chan<- error) {
	reportsDirectory := state.Global.GetReportsDirectory()
	lockFileDir := path.Join(reportsDirectory, fmt.Sprintf("%d", cycle))
	err := os.MkdirAll(lockFileDir, 0700)
	if err != nil {
		slog.Debug("failed to create lock file directory", "error", err.Error())
		resultChan <- err
		return
	}
	lockFilePath := path.Join(lockFileDir, ".lock")
	lock := filelock.NewLocker(lockFilePath)

	f, err := lock.Open()
	if err != nil {
		slog.Debug("failed to lock file", "error", err.Error())
		resultChan <- err
		return
	}
	slog.Debug("locked file", "file", lockFilePath)

	*unlockStore = func() error {
		err := f.Close()
		os.Remove(lockFilePath)
		return err
	}
	resultChan <- nil
}

func unlockFuncFactory(partialUnlocks []func() error) func() error {
	return func() error {
		var unlockErrors error
		for _, unlock := range partialUnlocks {
			if unlock == nil {
				continue
			}
			err := unlock()
			if err != nil {
				unlockErrors = errors.Join(unlockErrors, err)
			}
		}

		return unlockErrors
	}
}

func lockCycles(ctx context.Context, cycles ...int64) (unlock func() error, err error) {
	lockChannels := make([]chan error, len(cycles))
	unlockFunctions := make([]func() error, len(cycles))
	for i := range lockChannels {
		lockChannels[i] = make(chan error)
	}

	slog.Debug("locking cycles", "cycles", cycles)
	for i, cycle := range cycles {
		go lockCycle(cycle, &unlockFunctions[i], lockChannels[i])
	}

	allErrorLocksChannel := make(chan error)
	go func() {
		var lockErrors error
		for _, ch := range lockChannels {
			err := <-ch
			if err != nil {
				lockErrors = errors.Join(lockErrors, err)
			}
		}

		allErrorLocksChannel <- lockErrors
	}()

	unlock = unlockFuncFactory(unlockFunctions)

	select {
	case <-ctx.Done():
		slog.Debug("context canceled")
		return nil, ctx.Err()
	case lockErrors := <-allErrorLocksChannel:
		if lockErrors != nil {
			slog.Debug("failed to lock cycles", "error", lockErrors)
			unlock()
			return nil, lockErrors
		}

	}

	return unlock, nil
}

func lockCyclesWithTimeout(timeout time.Duration, cycles ...int64) (unlock func() error, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return lockCycles(ctx, cycles...)
}
