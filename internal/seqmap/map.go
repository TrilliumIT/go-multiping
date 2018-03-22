package seqmap

import (
	"errors"
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/ping"
)

type Map struct {
	l      sync.RWMutex
	m      map[uint16]*ping.Ping
	Handle func(*ping.Ping, error)
}

func New(h func(*ping.Ping, error)) *Map {
	return &Map{
		m:      make(map[uint16]*ping.Ping),
		Handle: h,
	}
}

var ErrSeqWrapped = errors.New("sequence wrapped")
var ErrDoesNotExist = errors.New("does not exist")

// the error indicates sequences wrapped, new ping is added and replaces old ping
func (s *Map) Add(p *ping.Ping) (int, error) {
	idx := uint16(p.ID)
	var l int
	var err error
	s.l.Lock()
	_, ok := s.m[idx]
	if ok {
		err = ErrSeqWrapped
	}
	s.m[idx] = p
	l = len(s.m)
	s.l.Unlock()
	return l, err
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
	s.l.Unlock()
	return p, l, err
}
