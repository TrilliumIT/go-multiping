package socket

import (
	"errors"
	"net"
	"sync"

	"github.com/TrilliumIT/go-multiping/pinger/internal/seqmap"
)

type IpIdMap struct {
	// returns the length of the map after modification
	//Add(net.IP, int) (*seqmap.SeqMap, int, error)
	// returns the length of the map after modification
	//Pop(net.IP, int) (*seqmap.SeqMap, int, error)
	//Get(net.IP, int) (*seqmap.SeqMap, bool)
	l sync.RWMutex
	m ipIdMap
}

func New(proto int) *IpIdMap {
	m := &IpIdMap{}
	switch proto {
	case 4:
		m.m = make(ip4IdMap)
	case 6:
		m.m = make(ip6IdMap)
	default:
		panic("invalid protocol")
	}
	return m
}

type ipIdMap interface {
	add(net.IP, int, *seqmap.SeqMap)
	del(net.IP, int)
	get(net.IP, int) (*seqmap.SeqMap, bool)
	length() int
}

var ErrAlreadyExists = errors.New("already exists")
var ErrDoesNotExist = errors.New("does not exist")

// returns the length of the map after modification
func (i *IpIdMap) Add(ip net.IP, id int) (sm *seqmap.SeqMap, l int, err error) {
	var ok bool
	i.l.Lock()
	sm, ok = i.m.get(ip, id)
	if ok {
		err = ErrAlreadyExists
	} else {
		sm = seqmap.New()
		i.m.add(ip, id, sm)
	}
	l = i.m.length()
	i.l.Unlock()
	return sm, l, err
}

func (i *IpIdMap) Pop(ip net.IP, id int) (sm *seqmap.SeqMap, l int, err error) {
	var ok bool
	i.l.Lock()
	sm, ok = i.m.get(ip, id)
	if !ok {
		err = ErrDoesNotExist
	}
	i.m.del(ip, id)
	l = i.m.length()
	i.l.Unlock()
	return sm, l, err
}

func (i *IpIdMap) Get(ip net.IP, id int) (sm *seqmap.SeqMap, ok bool) {
	i.l.RLock()
	sm, ok = i.m.get(ip, id)
	i.l.RUnlock()
	return sm, ok
}
