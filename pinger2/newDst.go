package pinger

import (
	"context"
	"time"
)

func newDst(ctx context.Context, cancel func()) *Dst {
	return &Dst{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (p *Pinger) NewDst() *Dst {
	return newDst(context.WithCancel(p.ctx))
}

func (p *Pinger) NewDstWithTimeout(timeout time.Duration) *Dst {
	return newDst(context.WithTimeout(p.ctx, timeout))
}

func (p *Pinger) NewDstWithDeadline(d time.Time) *Dst {
	return newDst(context.WithDeadline(p.ctx, d))
}

func NewDst() *Dst {
	return Default().NewDst()
}

func NewDstWithContext(ctx context.Context) *Dst {
	return newDst(context.WithCancel(ctx))
}

func NewDstWithTimeout(timeout time.Duration) *Dst {
	return Default().NewDstWithTimeout(timeout)
}

func NewDstWithDeadline(d time.Time) *Dst {
	return Default().NewDstWithDeadline(d)
}
