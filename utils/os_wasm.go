//go:build wasm

package utils

import (
	"context"
)

var (
	INTERUPT_CHANNEL = make(chan struct{}, 1)
)

func CallbackOnInterrupt(ctx context.Context, cb func()) {
	go func() {
		select {
		case <-INTERUPT_CHANNEL:
			cb()
		case <-ctx.Done():
		}
	}()
}

type ProtectedSection struct {
	info     string
	signaled bool
}

func NewProtectedSection(info string) *ProtectedSection {
	result := &ProtectedSection{
		info: info,
	}
	go func() {
		for {
			_, ok := <-INTERUPT_CHANNEL
			if !ok {
				break
			}
			result.signaled = true
		}
	}()

	return result
}

func StartNewProtectedSection(info string) *ProtectedSection {
	result := NewProtectedSection(info)
	result.Start()
	return result
}

func (p *ProtectedSection) Start() {
}

func (p *ProtectedSection) Resume() {
	p.Start()
}

func (p *ProtectedSection) Stop() {
}

func (p *ProtectedSection) Pause() {
	p.Stop()
}

func (p *ProtectedSection) Close() {
	p.Stop()
}

func (p *ProtectedSection) Signaled() bool {
	return p.signaled
}
