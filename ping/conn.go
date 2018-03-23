package ping

import (
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/conn"
	"github.com/TrilliumIT/go-multiping/ping/internal/endpointmap"
	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
	"github.com/TrilliumIT/go-multiping/ping/internal/timeoutmap"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Conn struct {
	dst     *net.IPAddr
	id      int
	timeout time.Duration
	s       *Socket
	// Count should only be accessed via atomic
	count int64
}

func (s *Socket) NewConn(dst *net.IPAddr, handle func(*Ping, error), timeout time.Duration) (*Conn, error) {
	iHandle := func(p *ping.Ping, err error) {
		handle(iPingToPing(p), err)
	}
	id, err := s.Add(dst, iHandle)
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
	c.s = nil // make anybody who tries to send after close panic
	return s.s.Del(c.dst, c.id)
}

// SendPing sends a ping, it returns the count
// Count is incremented whether or not the underlying send errors
// The sequence of the sent packet can be derived by casing count to uint16.
//
// The sent ping from this function will not point to the same object that will be returned to the handler
// on success or timeout
func (c *Conn) SendPing() (int, *Ping, error) {
	count := atomic.AddInt64(&c.count, 1)
	p := &ping.Ping{Dst: dst, ID: id, Seq: int(uint16(count)), TimeOut: timeout}
	err := c.s.s.SendPing(p)
	rp := iPingToPing(p)
	return count, rp, err
}

// Count returns the last sent count
func (c *Conn) Count() int {
	return int(atomic.LoadInt64(&c.count))
}

// Seq returns the last sent ICMP sequence number
func (c *Conn) Seq() int {
	return int(uint16(c.Count()))
}

// ID returns the ICMP ID associated with this connection
func (c *Conn) ID() int {
	return c.id
}
