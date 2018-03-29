package seqmap

import (
	"errors"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
)

type Map struct {
	l            sync.RWMutex
	m            map[uint16]*ping.Ping
	Handle       func(*ping.Ping, error)
	seqOffset    int
	fullWaiting  bool
	unfullNotify chan struct{}
	draining     chan struct{}
}

func New(h func(*ping.Ping, error)) *Map {
	return &Map{
		m:            make(map[uint16]*ping.Ping),
		Handle:       h,
		unfullNotify: make(chan struct{}),
	}
}

var ErrDoesNotExist = errors.New("does not exist")

func (s *Map) Add(p *ping.Ping) (length int) {
	var idx uint16
	s.l.Lock()
	for {
		if len(s.m) >= 1<<16 {
			s.fullWaiting = true
			s.l.Unlock()
			_, open := <-s.unfullNotify
			if !open {
				return 0
			}
			s.l.Lock()
			continue
		}
		idx = uint16(p.Count + s.seqOffset)
		_, ok := s.m[idx]
		if ok {
			s.seqOffset++
			continue
		}
		p.Seq = int(idx)
		s.m[idx] = p
		length = len(s.m)
		break
	}
	s.l.Unlock()
	return length
}

// returns done func to call after handler is compelete
func (s *Map) Pop(seq int) (*ping.Ping, int, func(), error) {
	idx := uint16(seq)
	var l int
	var err error
	done := func() {}
	s.l.Lock()
	p, ok := s.m[idx]
	if !ok {
		err = ErrDoesNotExist
	}
	delete(s.m, idx)
	l = len(s.m)
	if s.fullWaiting && l < 1<<16 {
		s.fullWaiting = false
		s.unfullNotify <- struct{}{}
	}
	if l == 0 && s.draining != nil {
		draining := s.draining
		done = func() { close(draining) }
	}
	s.l.Unlock()
	return p, l, done, err
}

func (s *Map) Close() {
	s.l.Lock()
	s.fullWaiting = false
	close(s.unfullNotify)
	s.l.Unlock()
}

func (s *Map) Drain() {
	s.l.Lock()
	if s.draining == nil {
		if len(s.m) == 0 {
			s.l.Unlock()
			return
		}
		s.draining = make(chan struct{})
	}
	draining := s.draining
	s.l.Unlock()
	<-draining
}
