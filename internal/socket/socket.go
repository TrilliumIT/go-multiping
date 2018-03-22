package socket

import (
	"net"
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/ping"
	"github.com/TrilliumIT/go-multiping/internal/conn"
	"github.com/TrilliumIT/go-multiping/internal/endpointmap"
	"github.com/TrilliumIT/go-multiping/internal/seqmap"
	"github.com/TrilliumIT/go-multiping/internal/timeoutmap"
)

type Socket struct {
	workers int
	buffer  int
	l       sync.RWMutex

	v4conn *conn.Conn
	v4em   *endpointmap.Map
	v4tm   *timeoutmap.Map

	v6conn *conn.Conn
	v6em   *endpointmap.Map
	v6tm   *timeoutmap.Map
}

func New(workers, buffer int) *Socket {
	s := &Socket{
		workers: workers,
		buffer:  buffer,

		v4em: endpointmap.New(4),
		v4tm: timeoutmap.New(4),

		v6em: endpointmap.New(6),
		v6tm: timeoutmap.New(6),
	}
	s.v4conn = conn.New(4, s.v4handle)
	s.v6conn = conn.New(4, s.v6handle)
	return s
}

func (s *Socket) getStuff(ip net.IP) (
	*conn.Conn, *endpointmap.Map, *timeoutmap.Map,
) {
	if ip.To4() == nil && ip.To16() != nil {
		return s.v6conn, s.v6em, s.v6tm
	}
	return s.v4conn, s.v4em, s.v4tm
}

func handle(
	em *endpointmap.Map, tm *timeoutmap.Map,
	rp *ping.Ping, err error,
) {
	sm, ok := em.Get(rp.Dst.IP, rp.ID)
	if !ok {
		return
	}
	sp, _, err := sm.Pop(rp.Seq)
	if err == seqmap.ErrDoesNotExist {
		return
	}
	tm.Del(rp.Dst.IP, rp.ID, rp.Seq)
	sp.UpdateFrom(rp)
	sm.Handle(sp, err)
}

func (s *Socket) v4handle(rp *ping.Ping, err error) {
	handle(s.v4em, s.v4tm, rp, err)
}

func (s *Socket) v6handle(rp *ping.Ping, err error) {
	handle(s.v6em, s.v6tm, rp, err)
}
