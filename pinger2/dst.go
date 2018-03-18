package pinger

import (
	"context"

	"github.com/TrilliumIT/go-multiping/ping"
)

type Dst struct {
	ctx    context.Context
	cancel func()

	// permanent set properties
	host string
	cb   func(*ping.Ping, error)

	// editable properties
	RetryOnResolveError bool
	RetryOnSendError    bool
	ReResolveEvery      int

	/*
		run function parameters
		id uint16
		count int

		dynamic function vars
		sent int
		id int

		pl      sync.Mutex
		pending map[uint16]*ping.Ping
	*/
}

func (d *Dst) Cancel() {
	d.cancel()
}
