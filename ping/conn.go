package ping

import (
	"math/rand"
	"net"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Conn struct {
	dst     *net.IPAddr
	id      int
	timeout time.Duration
	s       *Socket
	handle  func(*ping.Ping, error)
}

type HandleFunc func(*Ping, error)

func (s *Socket) NewConn(dst *net.IPAddr, handle HandleFunc, timeout time.Duration) (*Conn, error) {
	c := &Conn{
		dst:     dst,
		timeout: timeout,
		s:       s,
		handle:  func(p *ping.Ping, err error) { handle(iPingToPing(p), err) },
	}
	var err error
	c.id, err = s.s.Add(dst, c.handle)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Conn) Close() error {
	s := c.s
	c.s = nil // make anybody who tries to send after close panic
	return s.s.Del(c.dst, c.id)
}

// SendPing sends a ping, it returns the count
// Errors sending will be sent to the handler
// returns the count of the sent packet
func (c *Conn) SendPing() int {
	p := &ping.Ping{Dst: c.dst, ID: c.id, TimeOut: c.timeout}
	count, err := c.s.s.SendPing(p)
	if err != nil {
		c.handle(p, err)
	}
	return count
}

// ID returns the ICMP ID associated with this connection
func (c *Conn) ID() int {
	return c.id
}
