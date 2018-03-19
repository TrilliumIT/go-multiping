package ticker

import (
	"context"
	"time"
)

// Ticker is a ticker for an ICMP sequence
// C will fire either on the interval, or as soon as Cont is called followed by wait not blocking
type ManualTicker struct {
	ticker
	ready chan struct{}
	tick  chan struct{}
}

// Ready is to be called by pinger indicating that pinger is ready to recieve another tick
func (mt *ManualTicker) Ready() {
	mt.ready <- struct{}{}
}

func (mt *ManualTicker) Tick() {
	mt.tick <- struct{}{}
}

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
