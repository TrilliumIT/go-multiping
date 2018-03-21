package socket

import (
	"net"
	"sync"

	"github.com/TrilliumIT/go-multiping/pinger/internal/seqmap"
)

type ip6IdMap struct {
	l sync.RWMutex
	m map[[6]byte]*seqmap.SeqMap
}

func (i *ip6IdMap) Add(ip net.IP, id int) (*seqmap.SeqMap, int, error) {
	idx := toIP4Idx(ip, id)
	var l int
	var err error
	i.l.Lock()
	sm, ok := i.m[idx]
	if ok {
		err = ErrAlreadyExists
	} else {
		sm = seqmap.New()
		i.m[idx] = sm
	}
	l = len(i.m)
	i.l.Unlock()
	return sm, l, err
}

func (i *ip6IdMap) Pop(ip net.IP, id int) (*seqmap.SeqMap, int, error) {
	idx := toIP4Idx(ip, id)
	var l int
	var err error
	i.l.Lock()
	sm, ok := i.m[idx]
	if !ok {
		err = ErrDoesNotExist
	}
	delete(i.m, idx)
	l = len(i.m)
	i.l.Unlock()
	return sm, l, err
}

func (i *ip6IdMap) Get(ip net.IP, id int) (*seqmap.SeqMap, bool) {
	idx := toIP4Idx(ip, id)
	i.l.RLock()
	sm, ok := i.m[idx]
	i.l.RUnlock()
	return sm, ok
}
