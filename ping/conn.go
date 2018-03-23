package ping

import (
	"net"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
	"github.com/TrilliumIT/go-multiping/ping/internal/socket"
)

// IPConn holds a connection to a destination
type IPConn struct {
	dst     *net.IPAddr
	id      int
	timeout time.Duration
	s       *Socket
	handle  func(*ping.Ping, error)
}

// ErrNoIDs is returned when there are no icmp ids left to use
// Either you are trying to ping the same host with more than 2^16 connections
// or you are on windows and are running more than 2^16 connections total
var ErrNoIDs = socket.ErrNoIDs

// NewIPConn creates a new connection
func NewIPConn(dst *net.IPAddr, handle HandleFunc, timeout time.Duration) (*IPConn, error) {
	return DefaultSocket().NewIPConn(dst, handle, timeout)
}

// NewIPConn creates a new connection
func (s *Socket) NewIPConn(dst *net.IPAddr, handle HandleFunc, timeout time.Duration) (*IPConn, error) {
	return s.newIPConn(dst, iHandle(handle), timeout)
}

func (s *Socket) newIPConn(dst *net.IPAddr, handle func(*ping.Ping, error), timeout time.Duration) (*IPConn, error) {
	c := &IPConn{
		dst:     dst,
		timeout: timeout,
		s:       s,
		handle:  handle,
	}
	var err error
	c.id, err = s.s.Add(dst, c.handle)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *IPConn) Close() error {
	if c.s == nil {
		return nil
	}
	s := c.s
	c.s = nil // make anybody who tries to send after close panic
	return s.s.Del(c.dst.IP, c.id)
}

// SendPing sends a ping, it returns the count
// Errors sending will be sent to the handler
// returns the count of the sent packet
func (c *IPConn) SendPing() int {
	return c.sendPing(&ping.Ping{}, nil)
}

func (c *IPConn) sendPing(p *ping.Ping, err error) int {
	p.Dst, p.ID, p.TimeOut = c.dst, c.id, c.timeout
	if err != nil {
		c.handle(p, err)
		return 0
	}
	count, err := c.s.s.SendPing(p)
	if err != nil {
		c.handle(p, err)
	}
	return count
}

// ID returns the ICMP ID associated with this connection
func (c *IPConn) ID() int {
	return c.id
}
