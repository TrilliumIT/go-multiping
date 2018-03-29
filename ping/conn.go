package ping

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/conn"
	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
	"github.com/TrilliumIT/go-multiping/ping/internal/socket"
)

// IPConn holds a connection to a destination
type IPConn struct {
	count int64
	ipc   *ipConn
}

type ipConn struct {
	s       *Socket
	dst     *net.IPAddr
	id      int
	timeout time.Duration
	handle  func(*ping.Ping, error)
}

// ErrNoIDs is returned when there are no icmp ids left to use
// Either you are trying to ping the same host with more than 2^16 connections
// or you are on windows and are running more than 2^16 connections total
var ErrNoIDs = socket.ErrNoIDs

// ErrTimedOut is returned when a ping times out
var ErrTimedOut = socket.ErrTimedOut

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
		count: -1,
	}
	var err error
	c.ipc, err = s.newipConn(dst, handle, timeout)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Socket) newipConn(dst *net.IPAddr, handle func(*ping.Ping, error), timeout time.Duration) (*ipConn, error) {
	ipc := &ipConn{
		dst:     dst,
		timeout: timeout,
		s:       s,
		handle:  handle,
	}
	var err error
	ipc.id, err = s.s.Add(dst, ipc.handle)
	return ipc, err
}

func (c *IPConn) Close() error {
	return c.ipc.close()
}

func (c *ipConn) close() error {
	if c.s == nil {
		return nil
	}
	defer func() { c.s = nil }() // make anybody who tries to send after close panic
	return c.s.s.Del(c.dst.IP, c.id)
}

func (c *ipConn) drain() {
	if c.s == nil {
		return
	}
	c.s.s.Drain(c.dst.IP, c.id)
}

var ErrNotRunning = conn.ErrNotRunning

// SendPing sends a ping, it returns the count
// Errors sending will be sent to the handler
// returns the count of the sent packet
func (c *IPConn) SendPing() int {
	count := int(atomic.AddInt64(&c.count, 1))
	p := &ping.Ping{
		Count:   count,
		TimeOut: c.ipc.timeout,
		Sent:    time.Now(),
	}
	c.ipc.sendPing(p)
	return count
}

func (c *ipConn) sendPing(p *ping.Ping) {
	p.Dst, p.ID = c.dst, c.id
	err := c.s.s.SendPing(p)
	if err != nil {
		c.handle(p, err)
	}
	return
}

// ID returns the ICMP ID associated with this connection
func (c *IPConn) ID() int {
	return c.ipc.id
}
