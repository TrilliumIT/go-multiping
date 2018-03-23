package ping

import (
	"context"
	"net"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
)

// HostConn is an ICMP connection based on hostname
// which can be configured to re-resolve a host
type HostConn struct {
	s       *Socket
	host    string
	handle  HandleFunc
	dst     *net.IPAddr
	timeout time.Duration
}

//TODO host field in ping
func (s *Socket) NewHostConn(host string, handle HandleFunc, timeout time.Duration) *HostConn {
	return &HostConn{
		s:       s,
		host:    host,
		handle:  handle,
		timeout: timeout,
	}
}

// Returns true if the destination has changed or is unresolvable
func (h *HostConn) resolve() (changed bool, err error) {
	var dst *net.IPAddr
	dst, err = net.ResolveIPAddr("ip", h.host)
	changed = dst == nil || h.dst == nil || !dst.IP.Equal(h.dst.IP)
	return changed, err
}

func (h *HostConn) hostSend(reResolveEvery int, handler func(*ping.Ping, error)) (func() int, func() error) {
	var c *Conn
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
			c, err = h.s.newConn(h.dst, handler, h.timeout)
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

func (h *HostConn) Interval(ctx context.Context, count, reResolveEvery int, interval time.Duration) error {
	send, closeConn := h.hostSend(reResolveEvery, iHandle(h.handle))

	runInterval(ctx, send, count, interval)
	return closeConn()
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
