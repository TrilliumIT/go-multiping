package timeoutmap

import (
	"context"
	"net"
	"sync"
	"time"
)

type Map struct {
	l        sync.RWMutex
	to       tMap
	t        *time.Timer
	nextIP   net.IP
	nextID   int
	nextSeq  int
	nextTime time.Time
}

type tMap interface {
	add(net.IP, int, int, time.Time)
	del(net.IP, int, int)
	getNext() (net.IP, int, int, time.Time)
}

func New(proto int) *Map {
	m := &Map{
		t: time.NewTimer(time.Hour),
	}
	m.t.Stop()
	select {
	case <-m.t.C:
	default:
	}

	switch proto {
	case 4:
		m.to = make(ip4m)
	case 6:
		m.to = make(ip6m)
	default:
		panic("invalid protocol")
	}
	return m
}

func (m *Map) Add(ip net.IP, id, seq int, t time.Time) {
	m.l.Lock()
	m.to.add(ip, id, seq, t)
	m.l.Unlock()
	m.setNext()
}

func (m *Map) Del(ip net.IP, id, seq int) {
	m.l.Lock()
	m.to.del(ip, id, seq)
	m.l.Unlock()
	m.setNext()
}

func (m *Map) setNext() {
	m.l.RLock()
	pnt := m.nextTime
	m.nextIP, m.nextID, m.nextSeq, m.nextTime = m.to.getNext()
	if pnt != m.nextTime {
		m.t.Stop()
		select {
		case <-m.t.C:
		default:
		}
		if !m.nextTime.IsZero() {
			m.t.Reset(time.Until(m.nextTime))
		}
	}
	m.l.RUnlock()
}

func (m *Map) Next(ctx context.Context) (ip net.IP, id, seq int, t time.Time) {
	var tt time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case tt = <-m.t.C:
		}
		m.l.Lock()
		ip, id, seq, t = m.nextIP, m.nextID, m.nextSeq, m.nextTime
		if t.IsZero() || t.After(tt) {
			m.l.Unlock()
			continue
		}
		m.to.del(ip, id, seq)
		m.l.Unlock()
		m.setNext()
		return
	}
}
