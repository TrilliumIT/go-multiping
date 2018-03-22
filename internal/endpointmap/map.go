package endpointmap

import (
	"errors"
	"net"
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/ping"
	"github.com/TrilliumIT/go-multiping/internal/seqmap"
)

type Map struct {
	l sync.RWMutex
	m iMap
}

func New(proto int) *Map {
	m := &Map{}
	switch proto {
	case 4:
		m.m = make(ip4m)
	case 6:
		m.m = make(ip6m)
	default:
		panic("invalid protocol")
	}
	return m
}

type iMap interface {
	add(net.IP, int, *seqmap.Map)
	del(net.IP, int)
	get(net.IP, int) (*seqmap.Map, bool)
	length() int
}

var ErrAlreadyExists = errors.New("already exists")
var ErrDoesNotExist = errors.New("does not exist")

// returns the length of the map after modification
func (m *Map) Add(ip net.IP, id int, h func(*ping.Ping, error)) (sm *seqmap.Map, l int, err error) {
	var ok bool
	m.l.Lock()
	sm, ok = m.m.get(ip, id)
	if ok {
		err = ErrAlreadyExists
	} else {
		sm = seqmap.New(h)
		m.m.add(ip, id, sm)
	}
	l = m.m.length()
	m.l.Unlock()
	return sm, l, err
}

func (m *Map) Pop(ip net.IP, id int) (sm *seqmap.Map, l int, err error) {
	var ok bool
	m.l.Lock()
	sm, ok = m.m.get(ip, id)
	if !ok {
		err = ErrDoesNotExist
	}
	m.m.del(ip, id)
	l = m.m.length()
	m.l.Unlock()
	return sm, l, err
}

func (m *Map) Get(ip net.IP, id int) (sm *seqmap.Map, ok bool) {
	m.l.RLock()
	sm, ok = m.m.get(ip, id)
	m.l.RUnlock()
	return sm, ok
}
