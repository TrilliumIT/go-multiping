package socket

import (
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	"github.com/TrilliumIT/go-multiping/internal/ping"
	"github.com/TrilliumIT/go-multiping/internal/conn"
	"github.com/TrilliumIT/go-multiping/internal/endpointmap"
	"github.com/TrilliumIT/go-multiping/internal/timeoutmap"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Conn struct {
	dst     *net.IPAddr
	id      int
	timeout time.Duration
	s       *Socket
	count   int64
}

func (s *Socket) NewConn(dst *net.IPAddr, handle func(*ping.Ping, error), timeout time.Duration) (*Conn, error) {
	conn, em, tm := s.getStuff(dst.IP)
	id, err := s.add(conn, em, tm, dst, handle)
	if err != nil {
		return nil, err
	}
	c := &Conn{
		dst:     dst,
		id:      id,
		timeout: timeout,
		s:       s,
	}
	return c, nil
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
			err = conn.Run(s.workers, s.buffer)
		}
		break
	}
	return id, err
}

func (c *Conn) Close() error {
	s := c.s
	conn, em, tm := s.getStuff(c.dst.IP)
	c.s = nil // make anybody who tries to use conn after close panic
	return s.del(conn, em, tm, c.dst, c.id)
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

func (c *Conn) SendPing() error {
	conn, em, tm := c.s.getStuff(c.dst.IP)
	sm, ok := em.Get(c.dst.IP, c.id)
	if !ok {
		return endpointmap.ErrDoesNotExist
	}

	count := atomic.AddInt64(&c.count, 1)
	p := &ping.Ping{Dst: c.dst, ID: c.id, Seq: int(uint16(count)), TimeOut: c.timeout}

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
