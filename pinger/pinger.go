package pinger

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/TrilliumIT/go-multiping/internal/listenMap"
	"github.com/TrilliumIT/go-multiping/ping"
	"github.com/TrilliumIT/go-multiping/pinger/internal/pending"
	"github.com/TrilliumIT/go-multiping/pinger/internal/ticker"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// PingConf holds optional coniguration parameters
type PingConf struct {
	// Count is how many pings will be attempted
	// 0 for unlimited pings
	Count int

	// Interval is the interval between pings
	// 0 for flood ping, sending a new packet as soon as the previous one is replied or times out
	Interval time.Duration

	// Timeout is how long to wait before considering a ping timed out
	Timeout time.Duration

	// RandDelay causes the first ping to be delayed by a random amount up to interval
	RandDelay bool

	// RetryOnResolveError will cause callback to be called on resolution errors
	// Resolution errors can be identied by error type net.DNSError
	// Otherwise Ping() will stop and retun an error on resolution errors
	RetryOnResolveError bool

	// RetryOnSendError will cause callback to be called on send errors.
	// Otherwise Ping() will stop and return an error on send errors.
	RetryOnSendError bool

	// ReResolveEvery caused pinger to re-resolve dns for a host every n pings.
	// 0 to disable. 1 to re-resolve every ping
	ReResolveEvery int
}

// DefaultPingConf returns a default ping configuration with an interval and timeout of 1 second
func DefaultPingConf() *PingConf {
	return &PingConf{
		Interval: time.Second,
		Timeout:  time.Second,
	}
}

func (p *PingConf) validate() *PingConf {
	if p == nil {
		p = DefaultPingConf()
	}
	return p
}

// ErrTimedOut is returned when a ping is timed out
var ErrTimedOut = pending.ErrTimedOut

// ErrSeqWrapped is returned if a packet is still being waited on by the time the ping sequence number wrapped and sent a new ping with this same sequence number
// This is only likely to happen if using a very short interval with a very long, or non-existent timeout
var ErrSeqWrapped = errors.New("response not recieved before sequence wrapped")

// HandleFunc is a function to handle pings
type HandleFunc func(context.Context, *ping.Ping, error)

// Ping starts a ping using the global conn
// see Conn.Ping for details
func Ping(host string, hf HandleFunc, conf *PingConf) error {
	return DefaultConn().Ping(host, hf, conf)
}

// Once sends a single ping and returns it
func Once(host string, conf *PingConf) (*ping.Ping, error) {
	return DefaultConn().Once(host, conf)
}

// PingWithContext starts a ping using the global conn
// see Conn.PingWithContext for details
func PingWithContext(ctx context.Context, host string, hf HandleFunc, conf *PingConf) error {
	return DefaultConn().PingWithContext(ctx, host, hf, conf)
}

// Ping starts a ping. See PingWithContext for details
func (c *Conn) Ping(host string, hf HandleFunc, conf *PingConf) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return c.PingWithContext(ctx, host, hf, conf)
}

// Once sends a single ping and returns it
func (c *Conn) Once(host string, conf *PingConf) (*ping.Ping, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if conf == nil {
		conf = DefaultPingConf()
	}
	conf.Count = 1
	var p *ping.Ping
	hf := func(ctx context.Context, rp *ping.Ping, err error) {
		p = rp
	}
	err := c.PingWithContext(ctx, host, hf, conf)
	return p, err
}

// PingWithContext starts a new ping to host calling cb every time a reply is recieved or a packet times out
// Timed out packets will return an error of context.DeadlineExceeded to cb
// Ping will send up to count pings, or unlimited if count is 0
// Ping will send a ping each interval. If interval is 0 ping will flood ping, sending a new packet as soon
// as the previous one is returned or timed out
// Packets will be considered timed out after timeout. 0 will disable timeouts
// With a disabled, or very long timeout and a short interval the packet sequence may rollover and try to reuse
// a packet sequence before the last packet sent with that sequence is recieved. If this happens, the callback or the
// first packet will never be triggered
// GoRoutines will be leaked if context is never canceled
func (c *Conn) PingWithContext(ctx context.Context, host string, hf HandleFunc, conf *PingConf) error {
	conf = conf.validate()

	pm := pending.NewMap()
	pktWg := sync.WaitGroup{}

	var tick ticker.Ticker
	if conf.Interval > 0 {
		tick = ticker.NewIntervalTicker(conf.Interval, conf.RandDelay)
	} else {
		tick = ticker.NewFloodTicker(pktWg.Wait)
	}
	tickCtx, tickCancel := context.WithCancel(ctx)
	defer tickCancel()
	go tick.Run(tickCtx)
	tick.Ready()

	phf := func(p *ping.Ping, err error) { hf(ctx, p, err) }
	var id, seq uint16
	var dst *net.IPAddr
	sent := -1 // number of packets attempted to be sent
	var lCancel func()
	var err error
	for {
		sent++
		if conf.Count > 0 && sent >= conf.Count {
			break
		}

		select {
		case <-ctx.Done():
			run(lCancel)
			return nil
		case <-tick.C():
		}

		pktWg.Add(1)
		tick.Ready()

		if id == 0 {
			id = uint16(rand.Intn(1<<16-2) + 1)
		}
		seq = uint16(sent)

		p := &pending.Ping{P: &ping.Ping{
			Host:    host,
			ID:      int(id),
			Seq:     int(seq),
			TimeOut: conf.Timeout,
		}}
		p.Lock()
		var pCtx context.Context
		pCtx, p.Cancel = context.WithCancel(ctx)

		if dst == nil || (conf.ReResolveEvery != 0 && sent%conf.ReResolveEvery == 0) {
			var nDst *net.IPAddr
			nDst, p.Err = net.ResolveIPAddr("ip", host)
			if p.Err != nil {
				p.Unlock()
				p.Cancel()
				if conf.RetryOnResolveError {
					go p.Wait(pCtx, pm, phf, pktWg.Done)
					continue
				}
				pktWg.Done()
				run(lCancel)
				return p.Err
			}

			if dst == nil || !nDst.IP.Equal(dst.IP) { // IP changed
				if lCancel != nil { // cancel the current listener
					lCancel()
					lCancel = nil
				}
				dst = nDst
			}
		}
		p.P.Dst = dst.IP
		p.P.Src = c.lm.SrcAddr(dst.IP)

		if lCancel == nil { // we don't have a listner yet for this dst
			// Register with listenmap
			var lctx context.Context
			lctx, lCancel = context.WithCancel(ctx)
			err = c.lm.Add(lctx, dst.IP, id, pm.OnRecv)
			if err != nil {
				p.Unlock()
				p.Cancel()
				pktWg.Done() // we need to call done here, because we're not calling wait on this error. Add errors that arent ErrAlreadyExists are a returnable problem
				lCancel()
				lCancel = nil
				if err == listenMap.ErrAlreadyExists { // we already have this listener registered
					id = 0 // try a different id
					continue
				}
				return err
			}
		}

		if opp, ok := pm.Add(p); ok {
			// we've looped seq and this old pending packet is still hanging around, cancel it
			opp.SetError(ErrSeqWrapped)
			opp.Cancel()
		}

		p.Err = c.lm.Send(p.P, dst)

		if p.Err != nil {
			p.Unlock()
			p.Cancel()
			if conf.RetryOnSendError {
				go p.Wait(pCtx, pm, phf, pktWg.Done)
				continue
			}
			pktWg.Done()
			run(lCancel)
			return p.Err
		}

		if conf.Timeout > 0 {
			// we're not running wait yet, so nothing is waiting on this ctx, we're replacing it with one with a timeout now
			// but canceling is a good idea to release resources from the previous ctx
			p.Cancel()
			pCtx, p.Cancel = context.WithTimeout(ctx, conf.Timeout)
		}

		p.Unlock()
		go p.Wait(pCtx, pm, phf, pktWg.Done)
	}

	pktWg.Wait()
	run(lCancel)
	return nil
}

func run(f func()) {
	if f != nil {
		f()
	}
}
