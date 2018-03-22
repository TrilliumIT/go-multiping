package pinger

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
	id, err := s.Add(dst, handle)
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

func (c *Conn) Close() error {
	s := c.s
	c.s = nil // make anybody who tries to use conn after close panic
	return s.s.Del(c.dst, c.id)
}

// SendPing sends a ping, it returns the count
// Count is incremented whether or not the underlying send errors
// The sequence of the sent packet can be derived by casing count to uint16.
func (c *Conn) SendPing() (int, error) {
	count := atomic.AddInt64(&c.count, 1)
	seq := int(uint16(count))
	return count, s.s.SendPing(c.dst, c.id, seq, c.timeout)
}

func (c *Conn) Count() int {
	return int(atomic.LoadInt64(&c.count))
}
