package ticker

import (
	"context"
	"time"
)

// ManualTicker is a ticker that allows the caller to manually Tick()
type ManualTicker struct {
	ticker
	ready chan struct{}
	tick  chan struct{}
}

// Ready is to be called by pinger indicating that pinger is ready to recieve another tick
func (mt *ManualTicker) Ready() {
	mt.ready <- struct{}{}
}

// Tick ticks the manual ticker
func (mt *ManualTicker) Tick() {
	mt.tick <- struct{}{}
}

// NewManualTicker returns a new ManualTicker
func NewManualTicker() *ManualTicker {
	mt := newManualTicker()
	return &mt
}

func newManualTicker() ManualTicker {
	return ManualTicker{
		newTicker(),
		make(chan struct{}),
		make(chan struct{}),
	}
}

// Run runs the ticker. It will stop when context does
func (mt *ManualTicker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-mt.ready:
		}

	loop:
		for {
			select {
			case <-ctx.Done():
				return
			case <-mt.ready:
				continue loop
			case <-mt.tick:
			}

			select {
			case <-ctx.Done():
				return
			case mt.c <- time.Now():
				break loop
			case <-mt.ready:
			}
		}
	}
}
