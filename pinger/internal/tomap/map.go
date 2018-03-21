package tomap

import (
	"context"
	"net"
	"sync"
	"time"
)

type TimeoutMap struct {
	l        sync.RWMutex
	to       timeout
	t        *time.Timer
	nextIP   net.IP
	nextID   int
	nextSeq  int
	nextTime time.Time
}

type timeout interface {
	add(net.IP, int, int, time.Time)
	del(net.IP, int, int)
	getNext() (net.IP, int, int, time.Time)
}

func New(proto int) *TimeoutMap {
	tm := &TimeoutMap{
		t: time.NewTimer(time.Hour),
	}
	tm.t.Stop()
	select {
	case <-tm.t.C:
	default:
	}

	switch proto {
	case 4:
		tm.to = make(ip4ToM)
	case 6:
		tm.to = make(ip6ToM)
	default:
		panic("invalid protocol")
	}
	return tm
}

func (tm *TimeoutMap) Add(ip net.IP, id, seq int, t time.Time) {
	tm.l.Lock()
	tm.to.add(ip, id, seq, t)
	tm.l.Unlock()
	tm.setNext()
}

func (t *TimeoutMap) Del(ip net.IP, id, seq int) {
	t.l.Lock()
	t.to.del(ip, id, seq)
	t.l.Unlock()
	t.setNext()
}

func (t *TimeoutMap) setNext() {
	t.l.RLock()
	pnt := t.nextTime
	t.nextIP, t.nextID, t.nextSeq, t.nextTime = t.to.getNext()
	if pnt != t.nextTime {
		t.t.Stop()
		select {
		case <-t.t.C:
		default:
		}
		if !t.nextTime.IsZero() {
			t.t.Reset(time.Until(t.nextTime))
		}
	}
	t.l.RUnlock()
}

func (tm *TimeoutMap) Next(ctx context.Context) (ip net.IP, id, seq int, t time.Time) {
	var tt time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case tt = <-tm.t.C:
		}
		tm.l.Lock()
		ip, id, seq, t = tm.nextIP, tm.nextID, tm.nextSeq, tm.nextTime
		if t.IsZero() || t.After(tt) {
			tm.l.Unlock()
			continue
		}
		tm.to.del(ip, id, seq)
		tm.l.Unlock()
		tm.setNext()
	}
}
