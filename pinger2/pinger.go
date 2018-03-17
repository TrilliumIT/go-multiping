package pinger

import (
	"context"
	"sync"
)

type Pinger struct {
	ctx context.Context
}

var pinger *Pinger
var pingerLock sync.Mutex

func Default() *Pinger {
	pingerLock.Lock()
	defer pingerLock.Unlock()
	if pinger == nil {
		pinger = New()
	}
	return pinger
}

func New() *Pinger {
	return WithContext(context.Background())
}

func WithContext(ctx context.Context) *Pinger {
	p := &Pinger{
		ctx: ctx,
	}
	return p
}
