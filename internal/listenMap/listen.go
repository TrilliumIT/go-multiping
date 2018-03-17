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
type lmI [18]byte

// listen map entry
type lmE struct {
	cb  func(context.Context, *ping.Ping)
	ctx context.Context
}

func toLmI(ip net.IP, id int) lmI {
	var r lmI
	copy(r[0:16], ip.To16())
	binary.LittleEndian.PutUint16(r[16:], uint16(id))
	return r
}

func NewListenMap(ctx context.Context) *ListenMap {
	return &ListenMap{
		ctx: ctx,
		m:   make(map[lmI]*lmE),
		v4l: listener.New(4),
		v6l: listener.New(6),
	}
}

type ListenMap struct {
	m   map[lmI]*lmE
	l   sync.RWMutex
	ctx context.Context
	v4l *listener.Listener
	v6l *listener.Listener
}

func (l *ListenMap) getL(ip net.IP) *listener.Listener {
	if ip.To4() != nil {
		return l.v4l
	}
	return l.v6l
}

type alreadyExistsError struct{}

func (a *alreadyExistsError) Error() string {
	return "already exists"
}

func (lm *ListenMap) Add(ctx context.Context, ip net.IP, id int, cb func(context.Context, *ping.Ping)) error {
	return lm.add(ip, id, &lmE{cb, ctx})
}

func (lm *ListenMap) add(ip net.IP, id int, e *lmE) error {
	idx := toLmI(ip, id)
	err := lm.addIdx(idx, e)
	if err != nil {
		return err
	}
	l := lm.getL(ip)
	l.WgAdd(1)
	go func() {
		<-e.ctx.Done()
		lm.delIdx(idx)
		l.WgDone()
	}()

	if l.Running() {
		return nil
	}

	return l.Run(lm.GetCB)
}

func (l *ListenMap) addIdx(idx lmI, s *lmE) error {
	l.l.Lock()
	_, ok := l.m[idx]
	if ok {
		l.l.Unlock()
		return &alreadyExistsError{}
	}
	l.m[idx] = s
	l.l.Unlock()
	return nil
}

func (l *ListenMap) get(ip net.IP, id int) (*lmE, bool) {
	return l.getIdx(toLmI(ip, id))
}

func (lm *ListenMap) GetCB(ip net.IP, id int) func(context.Context, *ping.Ping) {
	lme, ok := lm.get(ip, id)
	if !ok {
		return nil
	}
	return lme.cb
}

func (l *ListenMap) getIdx(idx lmI) (*lmE, bool) {
	l.l.RLock()
	s, ok := l.m[idx]
	l.l.RUnlock()
	return s, ok
}

func (l *ListenMap) del(ip net.IP, id int) {
	l.delIdx(toLmI(ip, id))
}

func (l *ListenMap) delIdx(idx lmI) {
	l.l.Lock()
	delete(l.m, idx)
	l.l.Unlock()
}
