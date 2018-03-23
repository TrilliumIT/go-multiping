package ping

import (
	"context"
	"net"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
)

// HostConn is an ICMP connection based on hostname
// Pings run from a HostConn can be configured to periodically re-resolve
type HostConn struct {
	s       *Socket
	host    string
	handle  HandleFunc
	dst     *net.IPAddr
	timeout time.Duration
}

// NewHostConn returns a new HostConn
func NewHostConn(host string, handle HandleFunc, timeout time.Duration) *HostConn {
	return DefaultSocket().NewHostConn(host, handle, timeout)
}

// NewHostConn returns a new HostConn
func (s *Socket) NewHostConn(host string, handle HandleFunc, timeout time.Duration) *HostConn {
	return &HostConn{
		s:       s,
		host:    host,
		handle:  handle,
		timeout: timeout,
	}
}

// HostOnce pings a host once
func HostOnce(host string, timeout time.Duration) (*Ping, error) {
	return DefaultSocket().HostOnce(host, timeout)
}

// HostOnce sends a single echo request and returns, it blocks until a reply is recieved or the ping times out
//
// Zero is no timeout and Once will block forever if a reply is never recieved
//
// It is not recommended to use Once in a loop, use Interval instead
func (s *Socket) HostOnce(host string, timeout time.Duration) (*Ping, error) {
	sendGet := func(h HandleFunc) (func() int, func() error, error) {
		send := func() int {
			s.HostInterval(context.Background(), host, h, 1, 1, 0, timeout)
			return 1
		}
		return send, func() error { return nil }, nil
	}
	return runOnce(sendGet)
}

// Returns true if the destination has changed or is unresolvable
func (h *HostConn) resolve() (changed bool, err error) {
	var dst *net.IPAddr
	dst, err = net.ResolveIPAddr("ip", h.host)
	changed = dst == nil || h.dst == nil || !dst.IP.Equal(h.dst.IP)
	return changed, err
}

func (h *HostConn) hostSend(reResolveEvery int, handler func(*ping.Ping, error)) (func() int, func() error) {
	var c *IPConn
	var cCount int

	cClose := func() error {
		var err error
		if c != nil {
			err = c.Close()
		}
		return err
	}

	send := func() int {
		defer func() { cCount++ }()
		p := &ping.Ping{Host: h.host, TimeOut: h.timeout}
		changed := false
		var err error
		if h.dst == nil || cCount%reResolveEvery == 0 {
			changed, err = h.resolve()
			if err != nil {
				h.dst = nil
				if c != nil {
					_ = c.Close()
					c = nil
				}
				handler(p, err)
				return cCount
			}
		}
		if changed || c == nil {
			err = c.Close()
			if err != nil {
				c = nil
				handler(p, err)
				return cCount
			}
			c, err = h.s.newIPConn(h.dst, handler, h.timeout)
			if err != nil {
				c = nil
				handler(p, err)
				return cCount
			}
		}
		c.sendPing(p, nil)
		return cCount
	}
	return send, cClose
}

// HostInterval pings a host re-resolving the hostname every reResolveEvery pings
func HostInterval(ctx context.Context, host string, handle HandleFunc, count, reResolveEvery int, interval, timeout time.Duration) error {
	return DefaultSocket().HostInterval(ctx, host, handle, count, reResolveEvery, interval, timeout)
}

// HostInterval pings a host re-resolving the hostname every reResolveEvery pings
func (s *Socket) HostInterval(ctx context.Context, host string, handle HandleFunc, count, reResolveEvery int, interval, timeout time.Duration) error {
	h := s.NewHostConn(host, handle, timeout)
	return h.Interval(ctx, count, reResolveEvery, interval)
}

// Interval pings a host re-resolving the hostname every reResolveEvery pings
func (h *HostConn) Interval(ctx context.Context, count, reResolveEvery int, interval time.Duration) error {
	send, closeConn := h.hostSend(reResolveEvery, iHandle(h.handle))

	runInterval(ctx, send, count, interval)
	return closeConn()
}

// HostFlood pings a host re-resolving the hostname every reResolveEvery pings
func HostFlood(ctx context.Context, host string, handle HandleFunc, count, reResolveEvery int, timeout time.Duration) error {
	return DefaultSocket().HostFlood(ctx, host, handle, count, reResolveEvery, timeout)
}

// HostFlood pings a host re-resolving the hostname every reResolveEvery pings
func (s *Socket) HostFlood(ctx context.Context, host string, handle HandleFunc, count, reResolveEvery int, timeout time.Duration) error {
	h := s.NewHostConn(host, handle, timeout)
	return h.Flood(ctx, count, reResolveEvery)
}

// Flood continuously sends pings, sending the next ping as soon as the previous one is replied or times out.
func (h *HostConn) Flood(ctx context.Context, count, reResolveEvery int) error {
	fC := make(chan struct{})
	floodHander := func(p *ping.Ping, err error) {
		fC <- struct{}{}
		iHandle(h.handle)(p, err)
	}
	send, closeConn := h.hostSend(reResolveEvery, floodHander)

	runFlood(ctx, send, fC, count)
	return closeConn()
}
