package listenMap

import (
	"context"
	"encoding/binary"
	"net"
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/listenMap/internal/listener"
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

func NewListenMap(ctx context.Context) *ListenMap {
	return &ListenMap{
		m:   make(map[index]callback),
		v4l: listener.New(4),
		v6l: listener.New(6),
	}
}

type ListenMap struct {
	m   map[index]callback
	l   sync.RWMutex
	v4l *listener.Listener
	v6l *listener.Listener
}

func (l *ListenMap) getL(ip net.IP) *listener.Listener {
	if ip.To4() != nil {
		return l.v4l
	}
	return l.v6l
}

type ErrAlreadyExists struct{}

func (a *ErrAlreadyExists) Error() string {
	return "already exists"
}

func (lm *ListenMap) Send(p *ping.Ping, dst net.Addr) error {
	return lm.getL(p.Dst).Send(p, dst)
}

func (lm *ListenMap) SrcAddr(dst net.IP) net.IP {
	return lm.getL(dst).Props.SrcIP
}

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

	return l.Run(lm.GetCB)
}

func (l *ListenMap) addIdx(idx index, s callback) error {
	l.l.Lock()
	_, ok := l.m[idx]
	if ok {
		l.l.Unlock()
		return &ErrAlreadyExists{}
	}
	l.m[idx] = s
	l.l.Unlock()
	return nil
}

func (l *ListenMap) get(ip net.IP, id uint16) (callback, bool) {
	return l.getIdx(toIndex(ip, id))
}

func (lm *ListenMap) GetCB(ip net.IP, id uint16) func(context.Context, *ping.Ping) {
	lme, ok := lm.get(ip, id)
	if !ok {
		return nil
	}
	return lme
}

func (l *ListenMap) getIdx(idx index) (callback, bool) {
	l.l.RLock()
	s, ok := l.m[idx]
	l.l.RUnlock()
	return s, ok
}

func (l *ListenMap) del(ip net.IP, id uint16) {
	l.delIdx(toIndex(ip, id))
}

func (l *ListenMap) delIdx(idx index) {
	l.l.Lock()
	delete(l.m, idx)
	l.l.Unlock()
}
