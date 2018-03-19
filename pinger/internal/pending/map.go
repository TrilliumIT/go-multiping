package pending

import (
	"context"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
)

type Map struct {
	m map[uint16]*Ping
	l sync.Mutex
}

func NewMap() *Map {
	return &Map{
		m: make(map[uint16]*Ping),
	}
}

// Add returns the previous pendingPkt at this sequence if one existed
func (pm *Map) Add(pp *Ping) (*Ping, bool) {
	pm.l.Lock()
	opp, ok := pm.m[uint16(pp.P.Seq)]
	pm.m[uint16(pp.P.Seq)] = pp
	pm.l.Unlock()
	return opp, ok
}

// Del returns true if the item existed to be deleted
func (pm *Map) Del(seq uint16) bool {
	pm.l.Lock()
	_, ok := pm.m[seq]
	delete(pm.m, seq)
	pm.l.Unlock()
	return ok
}

func (pm *Map) Get(seq uint16) (*Ping, bool) {
	pm.l.Lock()
	opp, ok := pm.m[seq]
	pm.l.Unlock()
	return opp, ok
}

func (pm *Map) OnRecv(ctx context.Context, p *ping.Ping) {
	seq := uint16(p.Seq)
	pp, ok := pm.Get(seq)
	if !ok {
		return
	}

	pp.UpdateFrom(p)

	// cancel the timeout thread, will call cb and done() the waitgroup
	pp.Cancel()
}
