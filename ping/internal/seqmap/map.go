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
	count        int
	seqOffset    int
	fullWaiting  bool
	unfullNotify chan struct{}
}

func New(h func(*ping.Ping, error)) *Map {
	return &Map{
		m:            make(map[uint16]*ping.Ping),
		Handle:       h,
		unfullNotify: make(chan struct{}),
	}
}

var ErrDoesNotExist = errors.New("does not exist")

func (s *Map) Add(p *ping.Ping) (length, count int) {
	var idx uint16
	for {
		s.l.Lock()
		if len(s.m) == 1<<16 {
			s.fullWaiting = true
			s.l.Unlock()
			<-s.unfullNotify
			continue
		}
		idx = uint16(s.count + s.seqOffset)
		_, ok := s.m[idx]
		if ok {
			s.seqOffset++
			continue
		}
		p.Seq = int(idx)
		p.Count = s.count
		s.m[idx] = p
		count = s.count
		length = len(s.m)
		s.count++
		break
	}
	s.l.Unlock()
	return length, count
}

func (s *Map) Pop(seq int) (*ping.Ping, int, error) {
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
	s.l.Unlock()
	return p, l, err
}
