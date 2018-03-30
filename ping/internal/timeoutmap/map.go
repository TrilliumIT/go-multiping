package timeoutmap

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
)

type Map struct {
	l        sync.Mutex
	to       tMap
	t        *time.Timer
	nextIP   net.IP
	nextID   ping.ID
	nextSeq  ping.Seq
	nextTime time.Time
}

type tMap interface {
	add(net.IP, ping.ID, ping.Seq, time.Time)
	del(net.IP, ping.ID, ping.Seq)
	exists(net.IP, ping.ID, ping.Seq) bool
	getNext() (net.IP, ping.ID, ping.Seq, time.Time)
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

func (m *Map) Add(ip net.IP, id ping.ID, seq ping.Seq, t time.Time) {
	m.l.Lock()
	m.to.add(ip, id, seq, t)
	m.setNext()
	m.l.Unlock()
}

func (m *Map) Update(ip net.IP, id ping.ID, seq ping.Seq, t time.Time) {
	m.l.Lock()
	if m.to.exists(ip, id, seq) {
		m.to.add(ip, id, seq, t)
		m.setNext()
	}
	m.l.Unlock()
}

func (m *Map) Del(ip net.IP, id ping.ID, seq ping.Seq) {
	m.l.Lock()
	m.to.del(ip, id, seq)
	m.setNext()
	m.l.Unlock()
}

func (m *Map) setNext() {
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
}

func (m *Map) Next(ctx context.Context) (ip net.IP, id ping.ID, seq ping.Seq, t time.Time) {
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
		m.setNext()
		m.l.Unlock()
		return
	}
}
