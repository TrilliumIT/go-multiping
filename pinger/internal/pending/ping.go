package pending

import (
	"context"
	"errors"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
)

type Ping struct {
	Cancel func()
	P      *ping.Ping
	l      sync.Mutex
	Err    error
}

func (p *Ping) Lock() {
	p.l.Lock()
}

func (p *Ping) Unlock() {
	p.l.Unlock()
}

func (p *Ping) UpdateFrom(p2 *ping.Ping) {
	p.Lock()
	p.P.UpdateFrom(p2)
	p.Unlock()
}

func (p *Ping) SetError(err error) {
	p.Lock()
	p.Err = err
	p.Unlock()
}

var ErrTimedOut = errors.New("ping timed out")

func (p *Ping) Wait(ctx context.Context, pm *Map, cb func(*ping.Ping, error), done func()) {
	<-ctx.Done()
	p.l.Lock()
	pm.Del(uint16(p.P.Seq))
	if ctx.Err() == context.DeadlineExceeded && p.Err == nil {
		p.Err = ErrTimedOut
	}

	cb(p.P, p.Err)
	p.l.Unlock()

	done()
}
