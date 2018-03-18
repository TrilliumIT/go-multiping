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

type pendingMap struct {
	m map[uint16]*pendingPkt
	l sync.Mutex
}

// add returns the previous pendingPkt at this sequence if one existed
func (p *pendingMap) add(pp *pendingPkt) (*pendingPkt, bool) {
	p.l.Lock()
	opp, ok := p.m[uint16(pp.p.Seq)]
	p.m[uint16(pp.p.Seq)] = pp
	p.l.Unlock()
	return opp, ok
}

// del returns false if the item didn't exist
func (p *pendingMap) del(seq uint16) {
	p.l.Lock()
	delete(p.m, seq)
	p.l.Unlock()
}

func (p *pendingMap) get(seq uint16) (*pendingPkt, bool) {
	p.l.Lock()
	opp, ok := p.m[seq]
	p.l.Unlock()
	return opp, ok
}

func (pm *pendingMap) onRecv(ctx context.Context, p *ping.Ping) {
	seq := uint16(p.Seq)
	pp, ok := pm.get(seq)
	if !ok {
		return
	}

	pp.l.Lock()
	pp.p.UpdateFrom(p)
	pp.l.Unlock()

	// cancel the timeout thread, will call cb and done() the waitgroup
	pp.cancel()
}
