package pinger

import (
	"context"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
)

type PendingMap struct {
	m map[uint16]*PendingPing
	l sync.Mutex
}

// Add returns the previous pendingPkt at this sequence if one existed
func (p *PendingMap) Add(pp *PendingPing) (*PendingPing, bool) {
	p.l.Lock()
	opp, ok := p.m[uint16(pp.P.Seq)]
	p.m[uint16(pp.P.Seq)] = pp
	p.l.Unlock()
	return opp, ok
}

func (p *PendingMap) Del(seq uint16) {
	p.l.Lock()
	delete(p.m, seq)
	p.l.Unlock()
}

func (p *PendingMap) Get(seq uint16) (*PendingPing, bool) {
	p.l.Lock()
	opp, ok := p.m[seq]
	p.l.Unlock()
	return opp, ok
}

func (pm *PendingMap) OnRecv(ctx context.Context, p *ping.Ping) {
	seq := uint16(p.Seq)
	pp, ok := pm.Get(seq)
	if !ok {
		return
	}

	pp.l.Lock()
	pp.P.UpdateFrom(p)
	pp.l.Unlock()

	// cancel the timeout thread, will call cb and done() the waitgroup
	pp.Cancel()
}
