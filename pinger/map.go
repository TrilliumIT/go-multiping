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

func NewMap() *PendingMap {
	return &PendingMap{
		m: make(map[uint16]*PendingPing),
	}
}

// Add returns the previous pendingPkt at this sequence if one existed
func (pm *PendingMap) Add(pp *PendingPing) (*PendingPing, bool) {
	pm.l.Lock()
	opp, ok := pm.m[uint16(pp.P.Seq)]
	pm.m[uint16(pp.P.Seq)] = pp
	pm.l.Unlock()
	return opp, ok
}

func (pm *PendingMap) Del(seq uint16) {
	pm.l.Lock()
	delete(pm.m, seq)
	pm.l.Unlock()
}

func (pm *PendingMap) Get(seq uint16) (*PendingPing, bool) {
	pm.l.Lock()
	opp, ok := pm.m[seq]
	pm.l.Unlock()
	return opp, ok
}

func (pm *PendingMap) OnRecv(ctx context.Context, p *ping.Ping) {
	seq := uint16(p.Seq)
	pp, ok := pm.Get(seq)
	if !ok {
		return
	}

	pp.UpdateFrom(p)

	// cancel the timeout thread, will call cb and done() the waitgroup
	pp.Cancel()
}
