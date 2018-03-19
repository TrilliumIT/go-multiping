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
}

func (mt *ManualTicker) Ready() {
	select {
	case mt.ready <- struct{}{}:
		/*
			default:
				fmt.Println("runing ready default")
		*/
	}
}

// NewFloodTicker reutrns a new flood ticker.
// Wait should be a function that will block preventing the next tick from firing. A waitgroup for pending packets.
// Ready must be triggered before the flood ping will wait on wait(). This is necessary to prevent a race between
// waiting and adding to a waitgroup
func NewManualTicker() Ticker {
	mt := newManualTicker()
	return &mt
}

func newManualTicker() ManualTicker {
	return ManualTicker{
		newTicker(),
		make(chan struct{}),
	}
}

// Run runs the ticker. It will stop when context does
func (mt *ManualTicker) Run(ctx context.Context) {
	mt.run(ctx, func() {})
}

func (mt *ManualTicker) run(ctx context.Context, wait func()) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-mt.ready:
		}

	loop:
		for {
			wait()
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
