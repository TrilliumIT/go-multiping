package ticker

import (
	"context"
	"math/rand"
	"time"
)

// IntervalTicker is a ticker for an ICMP sequence
type IntervalTicker struct {
	ticker
	interval  time.Duration
	randDelay bool
}

// NewIntervalTicker returns a new ticker.
// Ready() is not used by interval tickers, they will tick on the interval no matter what.
// randDelay will cause the first tick to be delayed by a random interval up to interval
func NewIntervalTicker(interval time.Duration, randDelay bool) Ticker {
	return &IntervalTicker{
		newTicker(),
		interval,
		randDelay,
	}
}

// Run runs the ticker. It will stop when context does
func (it *IntervalTicker) Run(ctx context.Context) {
	if it.randDelay {
		ft := time.NewTimer(time.Duration(rand.Int63n(it.interval.Nanoseconds())))
		select {
		case <-ctx.Done():
			ft.Stop()
			return
		case it.c <- <-ft.C:
		}
		ft.Stop()
	}
	t := time.NewTicker(it.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case it.c <- <-t.C:
		}
	}
}
