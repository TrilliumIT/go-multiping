package pinger

import (
	"context"
	"encoding/binary"
	"net"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
)

// listen map index
type lmI [18]byte

// listen map entry
type lmE struct {
	cb  func(*ping.Ping)
	ctx context.Context
}

func toLmI(ip net.IP, id int) lmI {
	var r lmI
	copy(r[0:16], ip.To16())
	binary.LittleEndian.PutUint16(r[16:], uint16(id))
	return r
}

func newListenMap(ctx context.Context) *listenMap {
	return &listenMap{
		ctx: ctx,
		m:   make(map[lmI]*lmE),
		v4l: &listener{proto: 4},
		v6l: &listener{proto: 6},
	}
}

type listenMap struct {
	m   map[lmI]*lmE
	l   sync.RWMutex
	ctx context.Context
	v4l *listener
	v6l *listener
}

type listener struct {
	proto   int
	l       sync.RWMutex
	running bool
	wg      sync.WaitGroup
	ctx     context.Context
}

func (l *listener) run() {}

func (l *listenMap) getL(ip net.IP) *listener {
	if ip.To4() != nil {
		return l.v4l
	}
	return l.v6l
}

type alreadyExistsError struct{}

func (a *alreadyExistsError) Error() string {
	return "already exists"
}

func (lm *listenMap) add(ip net.IP, id int, s *lmE) error {
	idx := toLmI(ip, id)
	err := lm.addIdx(idx, s)
	if err != nil {
		return err
	}
	l := lm.getL(ip)
	l.wg.Add(1)
	go func() {
		<-s.ctx.Done()
		lm.delIdx(idx)
		l.wg.Done()
	}()

	l.l.RLock()
	r := l.running
	l.l.RUnlock()
	if r {
		return nil
	}

	l.l.Lock()
	if l.running {
		l.l.Unlock()
		return nil
	}

	l.running = true
	var cancel func()
	l.ctx, cancel = context.WithCancel(lm.ctx)
	l.run()
	l.l.Unlock()

	go func() {
		l.wg.Wait()
		cancel()
	}()
	return nil
}

func (l *listenMap) addIdx(idx lmI, s *lmE) error {
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

func (l *listenMap) get(ip net.IP, id int) (*lmE, bool) {
	return l.getIdx(toLmI(ip, id))
}

func (l *listenMap) getIdx(idx lmI) (*lmE, bool) {
	l.l.RLock()
	s, ok := l.m[idx]
	l.l.RUnlock()
	return s, ok
}

func (l *listenMap) del(ip net.IP, id int) {
	l.delIdx(toLmI(ip, id))
}

func (l *listenMap) delIdx(idx lmI) {
	l.l.Lock()
	delete(l.m, idx)
	l.l.Unlock()
}
