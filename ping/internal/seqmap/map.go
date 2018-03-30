package seqmap

import (
	"errors"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
)

type Map struct {
	l            sync.RWMutex
	m            map[ping.Seq]*ping.Ping
	handle       func(*ping.Ping, error)
	seqOffset    int
	fullWaiting  bool
	unfullNotify chan struct{}
	wg           sync.WaitGroup
}

func New(h func(*ping.Ping, error)) *Map {
	return &Map{
		m:            make(map[ping.Seq]*ping.Ping),
		handle:       h,
		unfullNotify: make(chan struct{}),
	}
}

func (m *Map) Handle(p *ping.Ping, err error) {
	m.handle(p, err)
	m.wg.Done()
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
		length = len(s.m)
		s.wg.Add(1) // wg.done is called after handler is run, so handler must be run for every pop
		break
	}
	s.l.Unlock()
	return length
}

func (s *Map) Pop(seq ping.Seq) (*ping.Ping, int, error) {
	idx := seq
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

func (s *Map) Close() {
	s.l.Lock()
	s.fullWaiting = false
	close(s.unfullNotify)
	s.l.Unlock()
}

// TODO // Drain gets called while handle is still running, causes drain to return while handle is processing and dropped packet!
func (s *Map) Drain() {
	s.wg.Wait()
}
