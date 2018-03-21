package seqmap

import (
	"errors"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
)

type SeqMap struct {
	l sync.RWMutex
	m map[uint16]*ping.Ping
}

func New() *SeqMap {
	return &SeqMap{m: make(map[uint16]*ping.Ping)}
}

var ErrSeqWrapped = errors.New("sequence wrapped")
var ErrDoesNotExist = errors.New("does not exist")

// takes a value to ensure that the caller can't modify what is in our map
// the error indicates sequences wrapped, new ping is added and replaces old ping
func (s *SeqMap) Add(p ping.Ping) (int, error) {
	idx := uint16(p.ID)
	var l int
	var err error
	s.l.Lock()
	_, ok := s.m[idx]
	if ok {
		err = ErrSeqWrapped
	}
	s.m[idx] = &p
	l = len(s.m)
	s.l.Unlock()
	return l, err
}

func (s *SeqMap) Pop(id int) (*ping.Ping, int, error) {
	idx := uint16(id)
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
