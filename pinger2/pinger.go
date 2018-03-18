package pinger

import (
	"context"
)

func init() {
	socketCreated = make(chan struct{})
}

type Pinger struct {
	ctx context.Context
}

var pinger *Pinger
var socketCreated chan struct{}

func Default() *Pinger {
	select {
	case <-socketCreated:
		return pinger
	default:
	}
	pinger = NewPinger()
	close(socketCreated)
	return pinger
}

func NewPinger() *Pinger {
	return WithContext(context.Background())
}

func WithContext(ctx context.Context) *Pinger {
	p := &Pinger{
		ctx: ctx,
	}
	return p
}
