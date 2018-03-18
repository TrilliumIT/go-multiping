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
	// Ctx is an optional context which can be used to cancel or timeout the ping operation
	Ctx context.Context

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
		Ctx:      context.Background(),
		Interval: time.Second,
		Timeout:  time.Second,
	}
}

func (p *PingConf) validate() *PingConf {
	if p == nil {
		p = DefaultPingConf()
	}
	if p.Ctx == nil {
		p.Ctx = context.Background()
	}
	return p
}

type pendingPkt struct {
	ctx    context.Context
	cancel func()
	p      *ping.Ping
}

// Ping starts a ping using the global conn
// see Conn.Ping for details
func Ping(host string, cb func(*ping.Ping, error), conf *PingConf) error {
	return DefaultConn().Ping(host, cb, conf)
}

// Ping starts a new ping to host calling cb every time a reply is recieved or a packet times out
// Timed out packets will return an error of context.DeadlineExceeded to cb
// Ping will send up to count pings, or unlimited if count is 0
// Ping will send a ping each interval. If interval is 0 ping will flood ping, sending a new packet as soon
// as the previous one is returned or timed out
// Packets will be considered timed out after timeout. 0 will disable timeouts
// With a disabled, or very long timeout and a short interval the packet sequence may rollover and try to reuse
// a packet sequence before the last packet sent with that sequence is recieved. If this happens, the callback or the
// first packet sent will be triggered with an error of context.Canceled
func (c *Conn) Ping(host string, cb func(*ping.Ping, error), conf *PingConf) error {
	conf = conf.validate()

	pendingLock := sync.Mutex{}
	pending := make(map[uint16]*pendingPkt)
	pktWg := sync.WaitGroup{}

	ctx, cancel := context.WithCancel(conf.Ctx)
	defer cancel()
	go func() {
		select {
		case <-c.ctx.Done():
			cancel()
		case <-ctx.Done():
		}
	}()

	intervalTick := make(chan time.Time)
	intervalTicker := func() {
		for {
			pktWg.Wait()
			select {
			case <-ctx.Done():
				return
			case intervalTick <- time.Now():
			}
		}
	}
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
					if conf.Interval == 0 {
						go func() { intervalTick <- time.Now() }()
					}
					continue
				}
				return err
			}

			// IP changed
			if !nDst.IP.Equal(dst.IP) {
				if lCancel != nil {
					lCancel()
					lCancel = nil
				}
				dst = nDst
			}
		}
		p.p.Dst = dst.IP
		p.p.Src = c.lm.SrcAddr(dst.IP)

		if lCancel == nil {
			// Register with listenmap
			var lctx context.Context
			lctx, lCancel = context.WithCancel(ctx)
			err = c.lm.Add(lctx, dst.IP, id, pendingCb)
			if err != nil {
				if _, ok := err.(*listenMap.ErrAlreadyExists); ok {
					id = 0
					lCancel = nil
					continue
				}
				return err
			}
		}

		// add to pending map
		if conf.Timeout > 0 {
			p.ctx, p.cancel = context.WithTimeout(ctx, conf.Timeout)
		} else {
			p.ctx, p.cancel = context.WithCancel(ctx)
		}
		pendingLock.Lock()
		opp, ok := pending[seq]
		pending[seq] = p
		if ok { // we've looped seq and this old pending packet is still hanging around, cancel it
			opp.cancel()
			go cb(opp.p, opp.ctx.Err())
		}
		pendingLock.Unlock()

		err = c.lm.Send(p.p, dst)
		if err != nil {
			pendingLock.Lock()
			delete(pending, seq)
			pendingLock.Unlock()
			if conf.RetryOnSendError {
				go cb(p.p, err)
				if conf.Interval == 0 {
					go func() { intervalTick <- time.Now() }()
				}
				continue
			}
			return err
		}

		pktWg.Add(1)
		go func() {
			defer pktWg.Done()
			<-p.ctx.Done()
			if p.ctx.Err() != context.DeadlineExceeded {
				return
			}

			pendingLock.Lock()
			_, ok := pending[seq]
			delete(pending, seq)
			pendingLock.Unlock()
			if !ok {
				return
			}
			cb(p.p, p.ctx.Err())
		}()
	}

	pktWg.Wait()
	return nil
}
