package pinger

import (
	"context"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
)

type pendingMap struct {
	m map[uint16]*PendingPing
	l sync.Mutex
}

// add returns the previous pendingPkt at this sequence if one existed
func (p *pendingMap) add(pp *PendingPing) (*PendingPing, bool) {
	p.l.Lock()
	opp, ok := p.m[uint16(pp.P.Seq)]
	p.m[uint16(pp.P.Seq)] = pp
	p.l.Unlock()
	return opp, ok
}

// del returns false if the item didn't exist
func (p *pendingMap) del(seq uint16) {
	p.l.Lock()
	delete(p.m, seq)
	p.l.Unlock()
}

func (p *pendingMap) get(seq uint16) (*PendingPing, bool) {
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
	pp.P.UpdateFrom(p)
	pp.l.Unlock()

	// cancel the timeout thread, will call cb and done() the waitgroup
	pp.Cancel()
}
