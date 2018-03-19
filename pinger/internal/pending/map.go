package pending

import (
	"context"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
)

// Map holds a map of pending icmps by sequence
type Map struct {
	m map[uint16]*Ping
	l sync.Mutex
}

// NewMap returns a new map
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

// Del deletes an item from the map
func (pm *Map) Del(seq uint16) {
	pm.l.Lock()
	delete(pm.m, seq)
	pm.l.Unlock()
}

// Get gets an entry from the map
func (pm *Map) Get(seq uint16) (*Ping, bool) {
	pm.l.Lock()
	opp, ok := pm.m[seq]
	pm.l.Unlock()
	return opp, ok
}

// OnRecv handles recieved packets, calling the handler in the map
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
