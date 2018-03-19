package ticker

import (
	"context"
	"time"
)

// Ticker is a ticker for an ICMP sequence
// C will fire either on the interval, or as soon as Cont is called followed by wait not blocking
type FloodTicker struct {
	ticker
	wait  func()
	ready chan struct{}
}

func (ft *FloodTicker) Ready() {
	ft.ready <- struct{}{}
}

// NewFloodTicker reutrns a new flood ticker.
// Wait should be a function that will block preventing the next tick from firing. A waitgroup for pending packets.
// Ready must be triggered before the flood ping will wait on wait(). This is necessary to prevent a race between
// waiting and adding to a waitgroup
func NewFloodTicker(wait func()) Ticker {
	return &FloodTicker{
		newTicker(),
		wait,
		make(chan struct{}),
	}
}

// Run runs the ticker. It will stop when context does
func (ft *FloodTicker) Run(ctx context.Context) {
	for {
		ft.wait()
		select {
		case <-ctx.Done():
			return
		case ft.c <- time.Now():
		}

		select {
		case <-ctx.Done():
			return
		case <-ft.ready:
		}
	}
}
