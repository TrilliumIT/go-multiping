package ping

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
)

// HostConn is an ICMP connection based on hostname
// Pings run from a HostConn can be configured to periodically re-resolve
type HostConn struct {
	s              *Socket
	ipc            *ipConn
	draining       []*ipConn
	host           string
	count          int64
	reResolveEvery int
	handle         func(*ping.Ping, error)
	timeout        time.Duration
}

// NewHostConn returns a new HostConn
func NewHostConn(host string, reResolveEvery int, handle HandleFunc, timeout time.Duration) *HostConn {
	return DefaultSocket().NewHostConn(host, reResolveEvery, handle, timeout)
}

// NewHostConn returns a new HostConn
func (s *Socket) NewHostConn(host string, reResolveEvery int, handle HandleFunc, timeout time.Duration) *HostConn {
	return s.newHostConn(host, reResolveEvery, iHandle(handle), timeout)
}

func (s *Socket) newHostConn(host string, reResolveEvery int, handle func(*ping.Ping, error), timeout time.Duration) *HostConn {
	return &HostConn{
		s:              s,
		host:           host,
		reResolveEvery: reResolveEvery,
		handle:         handle,
		timeout:        timeout,
		count:          -1,
	}
}

// SendPing sends a ping
func (h *HostConn) SendPing() int {
	count := int(atomic.AddInt64(&h.count, 1))
	p := &ping.Ping{
		Count:   count,
		Host:    h.host,
		TimeOut: h.timeout,
		Sent:    time.Now(),
	}
	if h.ipc == nil || (h.reResolveEvery != 0 && count%h.reResolveEvery == 0) {
		var dst *net.IPAddr
		dst, err := net.ResolveIPAddr("ip", h.host)
		changed := dst == nil || h.ipc == nil || h.ipc.dst == nil || !dst.IP.Equal(h.ipc.dst.IP)
		if err != nil {
			h.handle(p, err)
			return count
		}
		if changed {
			if h.ipc != nil {
				h.ipc.drain()
				h.draining = append(h.draining, h.ipc)
			}
			h.ipc, err = h.s.newipConn(dst, h.handle, h.timeout)
			if err != nil {
				h.handle(p, err)
				return count
			}
		}
	}
	h.ipc.sendPing(p)
	return count
}

func (h *HostConn) Close() error {
	for _, ipc := range h.draining {
		_ = ipc.close()
	}
	if h.ipc == nil {
		return nil
	}
	return h.ipc.close()
}

func HostOnce(host string, timeout time.Duration) (*Ping, error) {
	return DefaultSocket().HostOnce(host, timeout)
}

// HostOnce sends a single echo request and returns, it blocks until a reply is recieved or the ping times out
// Zero is no timeout and IPOnce will block forever if a reply is never recieved
// It is not recommended to use IPOnce in a loop, use Interval, or create a Conn and call SendPing() in a loop
func (s *Socket) HostOnce(host string, timeout time.Duration) (*Ping, error) {
	sendGet := func(hdl HandleFunc) (func() int, func() error, error) {
		h := s.NewHostConn(host, 1, hdl, timeout)
		return h.SendPing, h.Close, nil
	}
	return runOnce(sendGet)
}

func HostInterval(ctx context.Context, host string, reResolveEvery int, handler HandleFunc, count int, interval, timeout time.Duration) error {
	return DefaultSocket().HostInterval(ctx, host, reResolveEvery, handler, count, interval, timeout)
}

// HostInterval sends a ping each interval up to count pings or until ctx is canceled.
//
// If an interval of zero is specified, it will send pings as fast as possible.
// When there are 2^16 pending pings which have not received a reply, or timed out
// sending will block. This may be a limiting factor in how quickly pings can be sent.
//
// If a timeout of zero is specifed, pings will never time out.
//
// If a count of zero is specified, interval will continue to send pings until ctx is canceled.
func (s *Socket) HostInterval(ctx context.Context, host string, reResolveEvery int, handler HandleFunc, count int, interval, timeout time.Duration) error {
	h := s.NewHostConn(host, reResolveEvery, handler, timeout)

	runInterval(ctx, h.SendPing, count, interval)
	return h.Close()
}

func HostFlood(ctx context.Context, host string, reResolveEvery int, handler HandleFunc, count int, timeout time.Duration) error {
	return DefaultSocket().HostFlood(ctx, host, reResolveEvery, handler, count, timeout)
}

// IPFlood continuously sends pings, sending the next ping as soon as the previous one is replied or times out.
func (s *Socket) HostFlood(ctx context.Context, host string, reResolveEvery int, handler HandleFunc, count int, timeout time.Duration) error {
	fC := make(chan struct{})
	floodHander := func(p *Ping, err error) {
		fC <- struct{}{}
		handler(p, err)
	}

	h := s.NewHostConn(host, reResolveEvery, floodHander, timeout)

	runFlood(ctx, h.SendPing, fC, count)
	return h.Close()
}
