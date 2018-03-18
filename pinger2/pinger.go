package pinger

import (
	"context"
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
}

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

	pendingLock := sync.Mutex{}
	pending := make(map[uint16]*pendingPkt)
	pktWg := sync.WaitGroup{}

	intervalTick := make(chan time.Time)
	var intervalTicker func()
	if conf.Interval > 0 {
		intervalTicker = func() {
			intervalTicker := time.NewTicker(conf.Interval)
			defer intervalTicker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case intervalTick <- <-intervalTicker.C:
				}
			}
		}
	} else {
		intervalTicker = func() {
			for {
				pktWg.Wait()
				select {
				case <-ctx.Done():
					return
				case intervalTick <- time.Now():
				}
			}
		}
	}
	go intervalTicker()

	pendingCb := func(ctx context.Context, p *ping.Ping) {
		seq := uint16(p.Seq)
		pendingLock.Lock()
		pp, ok := pending[seq]
		delete(pending, seq)
		pendingLock.Unlock()
		if !ok {
			return
		}
		// cancel the timeout thread, which ends the waitgroup
		pp.cancel()
		pp.p.UpdateFrom(p)
		cb(pp.p, nil)
	}

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

		seq = uint16(sent)
		p := &pendingPkt{p: &ping.Ping{
			Host:    host,
			Seq:     int(seq),
			TimeOut: conf.Timeout,
		}}

		if id == 0 {
			id = uint16(rand.Intn(1<<16 - 1))
		}
		p.p.ID = int(id)

		if dst == nil || (conf.ReResolveEvery != 0 && sent%conf.ReResolveEvery == 0) {
			var nDst *net.IPAddr
			nDst, err = net.ResolveIPAddr("ip", host)
			if err != nil {
				if conf.RetryOnResolveError {
					go cb(p.p, err)
					dst = nil
					continue
				}
				return err
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
			err = c.lm.Add(lctx, dst.IP, id, pendingCb)
			if err == listenMap.ErrAlreadyExists { // we already have this listener registered
				id = 0 // try a different id
				lCancel = nil
				continue
			}
			if err != nil {
				return err
			}
		}

		var pCtx context.Context
		// add to pending map
		if conf.Timeout > 0 {
			pCtx, p.cancel = context.WithTimeout(ctx, conf.Timeout)
		} else {
			pCtx, p.cancel = context.WithCancel(ctx)
		}
		pendingLock.Lock()
		if opp, ok := pending[seq]; ok {
			// we've looped seq and this old pending packet is still hanging around, cancel it
			opp.cancel()
		}
		pending[seq] = p
		pendingLock.Unlock()

		pktWg.Add(1)
		err = c.lm.Send(p.p, dst)
		if err != nil {
			pendingLock.Lock()
			delete(pending, seq)
			pendingLock.Unlock()
			if conf.RetryOnSendError {
				go cb(p.p, err)
				pktWg.Done()
				continue
			}
			pktWg.Done()
			return err
		}

		go func() {
			defer pktWg.Done()
			<-pCtx.Done()
			if pCtx.Err() != context.DeadlineExceeded { // this was canceled, probably by being recieved
				return
			}

			pendingLock.Lock()
			_, ok := pending[seq]
			delete(pending, seq)
			pendingLock.Unlock()
			if !ok { // this packet was already removed from the map, maybe it was recieved just in time
				return
			}
			cb(p.p, pCtx.Err())
		}()
	}

	pktWg.Wait()
	return nil
}
