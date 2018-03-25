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
	draining     bool
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
	s.draining = false
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

func (s *Map) Pop(seq int) (*ping.Ping, int, bool, error) {
	idx := uint16(seq)
	var l int
	var err error
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
	draining := s.draining
	s.l.Unlock()
	return p, l, draining, err
}

func (s *Map) Close() {
	s.l.Lock()
	s.fullWaiting = false
	close(s.unfullNotify)
	s.l.Unlock()
}

func (s *Map) Drain() {
	s.l.Lock()
	s.draining = true
	s.l.Unlock()
}
