package ping

import (
	"net"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
	"github.com/TrilliumIT/go-multiping/ping/internal/socket"
)

// Conn holds a connection to a destination
type Conn struct {
	dst     *net.IPAddr
	id      int
	timeout time.Duration
	s       *Socket
	handle  func(*ping.Ping, error)
}

// HandleFunc is a function to handle responses
type HandleFunc func(*Ping, error)

func iHandle(handle HandleFunc) func(*ping.Ping, error) {
	return func(p *ping.Ping, err error) { handle(iPingToPing(p), err) }
}

// ErrNoIDs is returned when there are no icmp ids left to use
// Either you are trying to ping the same host with more than 2^16 connections
// or you are on windows and are running more than 2^16 connections total
var ErrNoIDS = socket.ErrNoIDs

// NewConn creates a new connection
func NewConn(dst *net.IPAddr, handle HandleFunc, timeout time.Duration) (*Conn, error) {
	return DefaultSocket().NewConn(dst, handle, timeout)
}

// NewConn creates a new connection
func (s *Socket) NewConn(dst *net.IPAddr, handle HandleFunc, timeout time.Duration) (*Conn, error) {
	c := &Conn{
		dst:     dst,
		timeout: timeout,
		s:       s,
		handle:  iHandle(handle),
	}
	var err error
	c.id, err = s.s.Add(dst, c.handle)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Conn) Close() error {
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
func (c *Conn) SendPing() int {
	return c.sendPing(&ping.Ping{}, nil)
}

func (c *Conn) sendPing(p *ping.Ping, err error) int {
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
func (c *Conn) ID() int {
	return c.id
}
