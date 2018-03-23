package ping

import (
	"context"
	"net"
	"time"
)

type ret struct {
	p   *Ping
	err error
}

func Once(dst *net.IPAddr, timeout time.Duration) (*Ping, error) {
	return DefaultSocket().Once(dst, timeout)
}

// Once sends a single echo request and returns, it blocks until a reply is recieved or the ping times out
// Zero is no timeout and Once will block forever if a reply is never recieved
// It is not recommended to use Once in a loop, use Interval, or create a Conn and call SendPing() in a loop
func (s *Socket) Once(dst *net.IPAddr, timeout time.Duration) (*Ping, error) {
	rCh := make(chan *ret)
	h := func(p *Ping, err error) {
		rCh <- &ret{p, err}
	}

	c, err := s.NewConn(dst, h, timeout)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	c.SendPing()
	r := <-rCh
	return r.p, r.err
}

func Interval(ctx context.Context, dst *net.IPAddr, handler HandleFunc, count int, interval, timeout time.Duration) error {
	return DefaultSocket().Interval(ctx, dst, handler, count, interval, timeout)
}

// Interval sends a ping each interval up to count pings or until ctx is canceled.
//
// If an interval of zero is specified, it will send pings as fast as possible.
// When there are 2^16 pending pings which have not received a reply, or timed out
// sending will block. This may be a limiting factor in how quickly pings can be sent.
//
// If a timeout of zero is specifed, pings will never time out.
//
// If a count of zero is specified, interval will continue to send pings until ctx is canceled.
func (s *Socket) Interval(ctx context.Context, dst *net.IPAddr, handler HandleFunc, count int, interval, timeout time.Duration) error {
	c, err := s.NewConn(dst, handler, timeout)
	if err != nil {
		return err
	}

	runInterval(ctx, c.SendPing, count, interval)
	return c.Close()
}

func runInterval(ctx context.Context, send func() int, count int, interval time.Duration) {
	var tC <-chan time.Time
	switch interval {
	case 0:
		tc := make(chan time.Time)
		tC = tc
		close(tc)
	default:
		t := time.NewTicker(interval)
		tC = t.C
		defer t.Stop()
	}

	for cCount := send(); cCount < count || count == 0; cCount = send() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		select {
		case <-ctx.Done():
			return
		case <-tC:
		}
	}
}

func Flood(ctx context.Context, dst *net.IPAddr, handler HandleFunc, count int, timeout time.Duration) error {
	return DefaultSocket().Flood(ctx, dst, handler, count, timeout)
}

// Flood continuously sends pings, sending the next ping as soon as the previous one is replied or times out.
func (s *Socket) Flood(ctx context.Context, dst *net.IPAddr, handler HandleFunc, count int, timeout time.Duration) error {
	fC := make(chan struct{})
	floodHander := func(p *Ping, err error) {
		fC <- struct{}{}
		handler(p, err)
	}

	c, err := s.NewConn(dst, floodHander, timeout)
	if err != nil {
		return err
	}

	runFlood(ctx, c.SendPing, fC, count)
	return c.Close()
}

func runFlood(ctx context.Context, send func() int, fC <-chan struct{}, count int) {
	for cCount := send(); cCount < count || count == 0; cCount = send() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		select {
		case <-ctx.Done():
			return
		case <-fC:
		}
	}
}
