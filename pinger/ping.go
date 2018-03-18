package pinger

import (
	"context"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
)

type PendingPing struct {
	Cancel func()
	P      *ping.Ping
	l      sync.Mutex
	Err    error
}

func (p *PendingPing) Lock() {
	p.l.Lock()
}

func (p *PendingPing) Unlock() {
	p.l.Unlock()
}

func (p *PendingPing) Wait(ctx context.Context, pm *pendingMap, cb func(*ping.Ping, error), done func()) {
	<-ctx.Done()
	p.l.Lock()
	pm.del(uint16(p.P.Seq))
	if ctx.Err() == context.DeadlineExceeded && p.Err == nil {
		p.Err = ErrTimedOut
	}

	cb(p.P, p.Err)
	p.l.Unlock()

	done()
}
