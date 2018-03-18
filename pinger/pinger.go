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

	// RandDelay causes the first ping to be delayed by a random amount up to interval
	RandDelay bool

	// Timeout is how long to wait before considering a ping timed out
	Timeout time.Duration

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

var ErrTimedOut = errors.New("ping timed out")
var ErrSeqWrapped = errors.New("response not recieved before sequence wrapped")

// Ping starts a ping using the global conn
// see Conn.Ping for details
func Ping(host string, cb func(*ping.Ping, error), conf *PingConf) error {
	return DefaultConn().Ping(host, cb, conf)
}

// PingWithContext starts a ping using the global conn
// see Conn.PingWithContext for details
func PingWithContext(ctx context.Context, host string, cb func(*ping.Ping, error), conf *PingConf) error {
	return DefaultConn().PingWithContext(ctx, host, cb, conf)
}

// Ping starts a ping. See PingWithContext for details
func (c *Conn) Ping(host string, cb func(*ping.Ping, error), conf *PingConf) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return c.PingWithContext(ctx, host, cb, conf)
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
func (c *Conn) PingWithContext(ctx context.Context, host string, cb func(*ping.Ping, error), conf *PingConf) error {
	conf = conf.validate()

	pm := &pendingMap{
		m: make(map[uint16]*pendingPkt),
		l: sync.Mutex{},
	}
	pktWg := sync.WaitGroup{}

	intervalTick := newIntervalTicker(conf.Interval, conf.RandDelay, pktWg.Wait)
	intervalCtx, intervalCancel := context.WithCancel(ctx)
	go intervalTick.run(intervalCtx)

	var id, seq uint16
	var dst *net.IPAddr
	var sent int = -1 // number of packets attempted to be sent
	var lCancel func()
	var err error
	for {
		sent++
		if conf.Count > 0 && sent >= conf.Count {
			break
		}

		select {
		case <-ctx.Done():
			return nil
		case <-intervalTick.C:
		}

		pktWg.Add(1)
		intervalTick.Cont()

		if id == 0 {
			id = uint16(rand.Intn(1<<16-2) + 1)
		}
		seq = uint16(sent)

		p := &pendingPkt{p: &ping.Ping{
			Host:    host,
			ID:      int(id),
			Seq:     int(seq),
			TimeOut: conf.Timeout,
		}}
		p.l.Lock()
		var pCtx context.Context
		pCtx, p.cancel = context.WithCancel(ctx)

		if dst == nil || (conf.ReResolveEvery != 0 && sent%conf.ReResolveEvery == 0) {
			var nDst *net.IPAddr
			nDst, p.err = net.ResolveIPAddr("ip", host)
			if p.err != nil {
				p.l.Unlock()
				p.cancel()
				if conf.RetryOnResolveError {
					go p.wait(pCtx, pm, cb, pktWg.Done)
					continue
				}
				pktWg.Done()
				return p.err
			}

			if dst == nil || !nDst.IP.Equal(dst.IP) { // IP changed
				if lCancel != nil { // cancel the current listener
					lCancel()
					lCancel = nil
				}
				dst = nDst
			}
		}
		p.p.Dst = dst.IP
		p.p.Src = c.lm.SrcAddr(dst.IP)

		if lCancel == nil { // we don't have a listner yet for this dst
			// Register with listenmap
			var lctx context.Context
			lctx, lCancel = context.WithCancel(ctx)
			err = c.lm.Add(lctx, dst.IP, id, pm.onRecv)
			if err != nil {
				p.l.Unlock()
				p.cancel()
				pktWg.Done() // we need to call done here, because we're not calling wait on this error. Add errors that arent ErrAlreadyExists are a returnable problem

				if err == listenMap.ErrAlreadyExists { // we already have this listener registered
					id = 0 // try a different id
					lCancel()
					lCancel = nil
					continue
				}
				return err
			}
		}

		if opp, ok := pm.add(p); ok {
			// we've looped seq and this old pending packet is still hanging around, cancel it
			opp.l.Lock()
			opp.err = ErrSeqWrapped
			opp.l.Unlock()
			opp.cancel()
		}

		p.err = c.lm.Send(p.p, dst)

		if p.err != nil {
			p.l.Unlock()
			p.cancel()
			if conf.RetryOnSendError {
				go p.wait(pCtx, pm, cb, pktWg.Done)
				continue
			}
			pktWg.Done()
			return p.err
		}

		if conf.Timeout > 0 {
			// we're not running wait yet, so nothing is waiting on this ctx, we're replacing it with one with a timeout now
			// but canceling is a good idea to release resources from the previous ctx
			p.cancel()
			pCtx, p.cancel = context.WithTimeout(ctx, conf.Timeout)
		}

		p.l.Unlock()
		go p.wait(pCtx, pm, cb, pktWg.Done)
	}

	intervalCancel()
	pktWg.Wait()
	return nil
}
