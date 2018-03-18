package pinger

import (
	"context"
	"math/rand"
	"time"
)

type intervalTicker struct {
	C         chan time.Time
	interval  time.Duration
	randDelay bool
	wait      func()
	Cont      func()
}

func newIntervalTicker(interval time.Duration, randDelay bool, wait func()) *intervalTicker {
	return &intervalTicker{
		C:         make(chan time.Time),
		interval:  interval,
		randDelay: randDelay,
		wait:      wait,
		Cont:      func() {},
	}
}

func (it *intervalTicker) run(ctx context.Context) {
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
