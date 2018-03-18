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

type pendingPkt struct {
	cancel func()
	p      *ping.Ping
	l      sync.Mutex
	err    error
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

	intervalTick := make(intervalTicker)
	intervalCtx, intervalCancel := context.WithCancel(ctx)
	go intervalTick.run(intervalCtx, conf.Interval, &pktWg)

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
		case <-intervalTick:
		}
		// pktWg.Add(1) this is done before intervalTick

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

			if !nDst.IP.Equal(dst.IP) { // IP changed
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

type pendingMap struct {
	m map[uint16]*pendingPkt
	l sync.Mutex
}

// add returns the previous pendingPkt at this sequence if one existed
func (p *pendingMap) add(pp *pendingPkt) (*pendingPkt, bool) {
	p.l.Lock()
	opp, ok := p.m[uint16(pp.p.Seq)]
	p.m[uint16(pp.p.Seq)] = pp
	p.l.Unlock()
	return opp, ok
}

// del returns false if the item didn't exist
func (p *pendingMap) del(seq uint16) {
	p.l.Lock()
	delete(p.m, seq)
	p.l.Unlock()
}

func (p *pendingMap) get(seq uint16) (*pendingPkt, bool) {
	p.l.Lock()
	opp, ok := p.m[seq]
	p.l.Unlock()
	return opp, ok
}

func (p *pendingPkt) wait(ctx context.Context, pm *pendingMap, cb func(*ping.Ping, error), done func()) {
	<-ctx.Done()
	p.l.Lock()
	pm.del(uint16(p.p.Seq))
	if ctx.Err() == context.DeadlineExceeded && p.err == nil {
		p.err = ErrTimedOut
	}

	cb(p.p, p.err)
	p.l.Unlock()

	done()
}

func (pm *pendingMap) onRecv(ctx context.Context, p *ping.Ping) {
	seq := uint16(p.Seq)
	pp, ok := pm.get(seq)
	if !ok {
		return
	}

	pp.l.Lock()
	pp.p.UpdateFrom(p)
	pp.l.Unlock()

	// cancel the timeout thread, will call cb and done() the waitgroup
	pp.cancel()
}

type intervalTicker chan time.Time

func (it intervalTicker) run(ctx context.Context, interval time.Duration, wg *sync.WaitGroup) {
	if interval > 0 {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case it <- <-t.C:
				wg.Add(1)
			}
		}
	}

	for {
		wg.Wait()
		select {
		case <-ctx.Done():
			return
		case it <- time.Now():
			wg.Add(1)
		}
	}
}
