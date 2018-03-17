package pinger

import (
	"context"

	"github.com/TrilliumIT/go-multiping/ping"
)

type Dst struct {
	ctx    context.Context
	cancel func()
	cb     func(*ping.Ping, error)
}

func (d *Dst) Cancel() {
	d.cancel()
}
