package socket

import (
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	"github.com/TrilliumIT/go-multiping/internal/conn"
	"github.com/TrilliumIT/go-multiping/internal/endpointmap"
	"github.com/TrilliumIT/go-multiping/internal/ping"
	"github.com/TrilliumIT/go-multiping/internal/timeoutmap"
)

func (s *Socket) Add(dst *net.IPAddr, handle func(*ping.Ping, error)) (int, error) {
	conn, em, tm := s.getStuff(dst.IP)
	return s.add(conn, em, tm, dst, handle)
}

func (s *Socket) add(
	conn *conn.Conn, em *endpointmap.Map, tm *timeoutmap.Map,
	dst *net.IPAddr, handle func(*ping.Ping, error),
) (int, error) {
	var id, sl int
	var err error
	s.l.Lock()
	defer s.l.Unlock()
	for {
		id = rand.Intn(1<<16-2) + 1
		_, sl, err = em.Add(dst.IP, id, handle)
		if err == endpointmap.ErrAlreadyExists {
			continue
		}
		if sl == 1 {
			err = conn.Run(s.Workers)
		}
		break
	}
	return id, err
}

func (s *Socket) Del(dst *net.IPAddr, id int) error {
	conn, em, tm := s.getStuff(c.dst.IP)
	return s.del(conn, em, tm, dst, id)
}

func (s *Socket) del(
	conn *conn.Conn, em *endpointmap.Map, tm *timeoutmap.Map,
	dst *net.IPAddr, id int) error {
	s.l.Lock()
	defer s.l.Unlock()
	_, sl, err := em.Pop(dst.IP, id)
	if err != nil {
		return err
	}
	if sl == 0 {
		err = conn.Stop()
	}
	return err
}

func (c *Socket) SendPing(
	dst *net.IPAddr, id int, seq int, timeout time.Duration,
) error {
	conn, em, tm := c.s.getStuff(dst.IP)
	sm, ok := em.Get(dst.IP, id)
	if !ok {
		return endpointmap.ErrDoesNotExist
	}

	p := &ping.Ping{Dst: dst, ID: id, Seq: seq, TimeOut: timeout}

	_, err := sm.Add(p)
	if err != nil {
		return err
	}
	err = conn.Send(p)
	if err != nil {
		_, _, _ = sm.Pop(p.Seq)
		return err
	}
	tm.Add(p.Dst.IP, p.ID, p.Seq, p.TimeOutTime())
	return nil
}
