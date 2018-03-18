package pinger

import (
	"context"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
)

type pendingPkt struct {
	cancel func()
	p      *ping.Ping
	l      sync.Mutex
	err    error
}

func (p *pendingPkt) wait(ctx context.Context, pm *pendingMap, cb func(*ping.Ping, error), done func()) {
	<-ctx.Done()
	p.l.Lock()
	pm.del(uint16(p.p.Seq))
	if ctx.Err() == context.DeadlineExceeded && p.err == nil {
		p.err = ErrTimedOut
	}

	cb(p.p, p.err)
	p.l.Unlock()

	done()
}
