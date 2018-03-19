package ticker

import (
	"context"
	"time"
)

// Ticker is a ticker for an ICMP sequence
// C will fire either when another ping should be sent
// Ready will be called by pinger when pinger is ready to send another packet
// Run starts the ticker
type Ticker interface {
	C() <-chan time.Time
	Ready()
	Run(context.Context)
}

type ticker struct {
	c chan time.Time
}

func (t *ticker) C() <-chan time.Time {
	return t.c
}

func (t *ticker) Ready() {}

func newTicker() ticker {
	return ticker{make(chan time.Time)}
}
