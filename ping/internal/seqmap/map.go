package seqmap

import (
	"errors"
	"fmt"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
)

type Map struct {
	l            sync.RWMutex
	m            map[ping.Seq]*ping.Ping
	Handle       func(*ping.Ping, error)
	seqOffset    int
	fullWaiting  bool
	unfullNotify chan struct{}
	draining     chan struct{}
}

func New(h func(*ping.Ping, error)) *Map {
	return &Map{
		m:            make(map[ping.Seq]*ping.Ping),
		Handle:       h,
		unfullNotify: make(chan struct{}),
	}
}

var ErrDoesNotExist = errors.New("does not exist")

func (s *Map) Add(p *ping.Ping) (length int) {
	var idx ping.Seq
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
		idx = ping.Seq(p.Count + s.seqOffset)
		_, ok := s.m[idx]
		if ok {
			s.seqOffset++
			continue
		}
		p.Seq = idx
		s.m[idx] = p
		if s.draining != nil {
			panic("add after draining")
		}
		length = len(s.m)
		fmt.Printf("added %p to %p\n", p, s)
		break
	}
	s.l.Unlock()
	return length
}

// returns done func to call after handler is compelete
func (s *Map) Pop(seq ping.Seq) (*ping.Ping, int, func(), error) {
	idx := seq
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
		fmt.Printf("Popped %p from %p, returning done\n", p, s)
		draining := s.draining
		done = func() { close(draining) }
	} else {
		if l == 0 {
			fmt.Printf("Popped %p from %p, no drain requested\n", p, s)
		} else {
			fmt.Printf("Popped %p from %p\n", p, s)
		}
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

// TODO // Drain gets called while handle is still running, causes drain to return while handle is processing and dropped packet!
func (s *Map) Drain() {
	s.l.Lock()
	if s.draining == nil {
		s.draining = make(chan struct{})
		if len(s.m) == 0 {
			fmt.Printf("drain requested from %p, already 0\n", s)
			close(s.draining)
		} else {
			fmt.Printf("drain requested from %p\n", s)
		}
	}
	draining := s.draining
	s.l.Unlock()
	<-draining
}
