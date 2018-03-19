package listenmap

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/listenmap/internal/listener"
	"github.com/TrilliumIT/go-multiping/ping"
)

// listen map index
type index [18]byte

// listen map entry
type callback func(context.Context, *ping.Ping)

func toIndex(ip net.IP, id uint16) index {
	var r index
	copy(r[0:16], ip.To16())
	binary.LittleEndian.PutUint16(r[16:], id)
	return r
}

// NewListenMap returns a new listenmap
func NewListenMap() *ListenMap {
	return &ListenMap{
		m:   make(map[index]callback),
		v4l: listener.New(4),
		v6l: listener.New(6),
	}
}

// ListenMap provides protected access to a map of returning addresses and icmp ID to callbacks
type ListenMap struct {
	m       map[index]callback
	l       sync.RWMutex
	v4l     *listener.Listener
	v6l     *listener.Listener
	workers int
	buffer  int
}

func (lm *ListenMap) getL(ip net.IP) *listener.Listener {
	if ip.To4() != nil {
		return lm.v4l
	}
	return lm.v6l
}

// ErrAlreadyExists is returned when an entry for an ip/id combination already exists
var ErrAlreadyExists = errors.New("listener already exists")

// Send sends a ping to dst
func (lm *ListenMap) Send(p *ping.Ping, dst net.Addr) error {
	return lm.getL(p.Dst).Send(p, dst)
}

// SrcAddr returns the source address, 0.0.0.0 or ::
func (lm *ListenMap) SrcAddr(dst net.IP) net.IP {
	return lm.getL(dst).Props.SrcIP
}

// Add adds an ip/id pair to the map. It is removed when ctx is canceled
func (lm *ListenMap) Add(ctx context.Context, ip net.IP, id uint16, cb func(context.Context, *ping.Ping)) error {
	return lm.add(ctx, ip, id, cb)
}

func (lm *ListenMap) add(ctx context.Context, ip net.IP, id uint16, cb callback) error {
	idx := toIndex(ip, id)
	err := lm.addIdx(idx, cb)
	if err != nil {
		return err
	}
	l := lm.getL(ip)
	l.WgAdd(1)
	go func() {
		<-ctx.Done()
		lm.delIdx(idx)
		l.WgDone()
	}()

	if l.Running() {
		return nil
	}

	return l.Run(lm.GetCB, lm.workers, lm.buffer)
}

// SetWorkers sets the number of workers for a given listener
func (lm *ListenMap) SetWorkers(n int) {
	lm.workers = n
}

// SetBuffer sets the size of the buffer for the listener
func (lm *ListenMap) SetBuffer(n int) {
	lm.buffer = n
}

func (lm *ListenMap) addIdx(idx index, s callback) error {
	lm.l.Lock()
	_, ok := lm.m[idx]
	if ok {
		lm.l.Unlock()
		return ErrAlreadyExists
	}
	lm.m[idx] = s
	lm.l.Unlock()
	return nil
}

func (lm *ListenMap) get(ip net.IP, id uint16) (callback, bool) {
	return lm.getIdx(toIndex(ip, id))
}

// GetCB gets the callback for a given ip/id
func (lm *ListenMap) GetCB(ip net.IP, id uint16) func(context.Context, *ping.Ping) {
	lme, ok := lm.get(ip, id)
	if !ok {
		return nil
	}
	return lme
}

func (lm *ListenMap) getIdx(idx index) (callback, bool) {
	lm.l.RLock()
	s, ok := lm.m[idx]
	lm.l.RUnlock()
	return s, ok
}

func (lm *ListenMap) delIdx(idx index) {
	lm.l.Lock()
	delete(lm.m, idx)
	lm.l.Unlock()
}
