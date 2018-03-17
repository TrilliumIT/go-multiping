package pinger

import (
	"context"
	"encoding/binary"
	"net"
	"sync"

	"golang.org/x/net/icmp"

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
	proto int
	l     sync.RWMutex
	dead  chan struct{}
	wg    sync.WaitGroup
	ctx   context.Context
	conn  *icmp.PacketConn
}

func (l *listener) running() bool {
	l.l.RLock()
	r := l.usRunning()
	l.l.RUnlock()
	return r
}

func (l *listener) usRunning() bool {
	select {
	case <-l.dead:
		return false
	default:
	}
	return true
}

func (l *listener) run() error {
	l.dead = make(chan struct{})
	var err error

	go func() {
		defer close(l.dead)
	}()
	return nil
}

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

	if l.running() {
		return nil
	}

	l.l.Lock()
	if l.usRunning() {
		l.l.Unlock()
		return nil
	}

	var cancel func()
	l.ctx, cancel = context.WithCancel(lm.ctx)
	l.run()
	l.l.Unlock()

	go func() {
		l.wg.Wait()
		l.l.Lock()
		cancel()
		<-l.dead
		l.l.Unlock()
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
