package socket

import (
	"net"

	"github.com/TrilliumIT/go-multiping/ping"
	"github.com/TrilliumIT/go-multiping/pinger/internal/conn"
	"github.com/TrilliumIT/go-multiping/pinger/internal/endpointmap"
	"github.com/TrilliumIT/go-multiping/pinger/internal/seqmap"
	"github.com/TrilliumIT/go-multiping/pinger/internal/timeoutmap"
)

type Socket struct {
	v4conn *conn.Conn
	v4em   *endpointmap.Map
	v4tm   *timeoutmap.Map

	v6conn *conn.Conn
	v6em   *endpointmap.Map
	v6tm   *timeoutmap.Map
}

func New() *Socket {
	s := &Socket{
		v4em: endpointmap.New(4),
		v4tm: timeoutmap.New(4),

		v6em: endpointmap.New(6),
		v6tm: timeoutmap.New(6),
	}
	s.v4conn = conn.New(4, s.v4handle)
	s.v6conn = conn.New(4, s.v6handle)
	return s
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

func (s *Socket) Add(ip net.IP) {
}
