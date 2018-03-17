package pinger

import (
	"context"
	"time"

	"github.com/TrilliumIT/go-multiping/ping"
)

func newDst(ctx context.Context, cancel func()) *Dst {
	return &Dst{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (d *Dst) setCallback(cb func(*ping.Ping, error)) *Dst {
	d.cb = cb
	return d
}

func (p *Pinger) NewDst(cb func(*ping.Ping, error)) *Dst {
	return newDst(context.WithCancel(p.ctx)).setCallback(cb)
}

func (p *Pinger) NewDstWithTimeout(cb func(*ping.Ping, error), timeout time.Duration) *Dst {
	return newDst(context.WithTimeout(p.ctx, timeout)).setCallback(cb)
}

func (p *Pinger) NewDstWithDeadline(cb func(*ping.Ping, error), d time.Time) *Dst {
	return newDst(context.WithDeadline(p.ctx, d)).setCallback(cb)
}

func NewDst(cb func(*ping.Ping, error)) *Dst {
	return Default().NewDst(cb)
}

func NewDstWithContext(ctx context.Context, cb func(*ping.Ping, error)) *Dst {
	return newDst(context.WithCancel(ctx)).setCallback(cb)
}

func NewDstWithTimeout(cb func(*ping.Ping, error), timeout time.Duration) *Dst {
	return Default().NewDstWithTimeout(cb, timeout)
}

func NewDstWithDeadline(cb func(*ping.Ping, error), d time.Time) *Dst {
	return Default().NewDstWithDeadline(cb, d)
}
