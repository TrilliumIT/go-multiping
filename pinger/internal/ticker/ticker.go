package ticker

import (
	"context"
	"math/rand"
	"time"
)

// Ticker is a ticker for an ICMP sequence
// C will fire either on the interval, or as soon as Cont is called followed by wait not blocking
type Ticker struct {
	C         chan time.Time
	interval  time.Duration
	randDelay bool
	wait      func()
	Cont      func()
}

// NewTicker reutrns a new ticker. Wait should be a function that will block preventing the next tick from firing
// wait is only important for flood pings (interval == 0)
func NewTicker(interval time.Duration, randDelay bool, wait func()) *Ticker {
	return &Ticker{
		C:         make(chan time.Time),
		interval:  interval,
		randDelay: randDelay,
		wait:      wait,
		Cont:      func() {},
	}
}

// Run runs the ticker. It will stop when context does
func (it *Ticker) Run(ctx context.Context) {
	if it.interval > 0 {
		if it.randDelay {
			ft := time.NewTimer(time.Duration(rand.Int63n(it.interval.Nanoseconds())))
			select {
			case <-ctx.Done():
				ft.Stop()
				return
			case it.C <- <-ft.C:
			}
			ft.Stop()
		}
		t := time.NewTicker(it.interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case it.C <- <-t.C:
			}
		}
	}

	cont := make(chan struct{})
	it.Cont = func() { cont <- struct{}{} }
	for {
		it.wait()
		select {
		case <-ctx.Done():
			return
		case it.C <- time.Now():
		}

		select {
		case <-ctx.Done():
			return
		case <-cont:
		}
	}
}
