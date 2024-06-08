package utils

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func CallbackOnInterrupt(ctx context.Context, cb func()) {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-c:
			cb()
		case <-ctx.Done():
		}
		signal.Stop(c)
	}()
}

type ProtectedSection struct {
	info     string
	ch       chan os.Signal
	signaled bool
}

// Creates a new ProtectedSection
// Protected section captures SIGINT and SIGTERM signals if active
func NewProtectedSection(info string) *ProtectedSection {
	result := &ProtectedSection{
		ch:   make(chan os.Signal, 1),
		info: info,
	}
	// goroutine to handle signals and log details
	go func() {
		for {
			sig, ok := <-result.ch
			if !ok {
				break
			}
			result.signaled = true
			slog.Warn("received signal", "signal", sig, "info", info)
		}
	}()

	return result
}

// Creates a new ProtectedSection and starts it
// Protected section captures SIGINT and SIGTERM signals
func StartNewProtectedSection(info string) *ProtectedSection {
	result := NewProtectedSection(info)
	result.Start()
	return result
}

func (p *ProtectedSection) Start() {
	signal.Notify(p.ch, syscall.SIGINT, syscall.SIGTERM)
}

func (p *ProtectedSection) Resume() {
	p.Start()
}

func (p *ProtectedSection) Stop() {
	signal.Stop(p.ch)
}

func (p *ProtectedSection) Pause() {
	p.Stop()
}

func (p *ProtectedSection) Close() {
	p.Stop()
	close(p.ch)
}

func (p *ProtectedSection) Signaled() bool {
	return p.signaled
}
