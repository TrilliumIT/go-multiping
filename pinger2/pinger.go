package pinger

import (
	"context"
)

func init() {
	pingerCreated = make(chan struct{})
}

type Pinger struct {
	ctx context.Context
}

var pinger *Pinger
var pingerCreated chan struct{}

func Default() *Pinger {
	select {
	case <-pingerCreated:
		return pinger
	default:
	}
	pinger = NewPinger()
	close(pingerCreated)
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
