package pinger

import (
	"sync"
	"time"

	"github.com/TrilliumIT/go-multiping/ping"
)

// Dst is a destination host to be pinged
type Dst struct {
	host string
	// Interval is the ping interval
	interval time.Duration
	// Timeout is how long to wait after the last packet is sent
	timeout time.Duration
	// Count is the count of pings to be sent
	count    int
	callBack func(*ping.Ping, error)

	// randDelay controls whether the first ping will be dalayed by a random amount of time between 0 and interval
	randDelay      bool
	reResolve      bool
	reSend         bool
	callBackOnSend bool
	// callbacks
	/*
		onReply     func(*ping.Ping)
		onSend      func(*ping.Ping)
		onSendError func(*ping.Ping, error)
		onTimeout   func(*ping.Ping)
	*/

	pinger  *Pinger
	stop    chan struct{}
	sending chan struct{}
	cbWg    sync.WaitGroup
	pktCh   chan *pkt
	cbCh    chan *pktWErr
}

// NewDst creates a Dst
func NewDst(host string, interval, timeout time.Duration, count int, callBack func(*ping.Ping, error)) *Dst {
	return getGlobalPinger().NewDst(host, interval, timeout, count, callBack)
}

// NewDst creates a Dst
func (p *Pinger) NewDst(host string, interval, timeout time.Duration, count int, callBack func(*ping.Ping, error)) *Dst {
	return &Dst{
		host:     host,
		interval: interval,
		timeout:  timeout,
		count:    count,
		pinger:   p,
		callBack: callBack,
		stop:     make(chan struct{}),
	}
}

// EnableReResolve enables re-resolving the host before each ping. If this is not set, the host is only resolved once at the beginning of Run(). If an error occurs at this time, it is returned to Run().
// If this is set, CallBack will be called and err will include the resolve error any time resolve fails
// DNS errors can be identified as type *net.DNSError
func (d *Dst) EnableReResolve() {
	d.cbWg.Wait()
	d.reResolve = true
}

// EnableReSend enables resending ping that fail to send. If this is not set, send errors will be returned to Run().
// If this is set, CallBack will be called and err will include the resolve error any time resolve fails
// On linux, sending a ping to a non-routable ip, or an ip blocked by iptables
// will result in a send error.
// Windows does not properly report send errors for non-routable ips
func (d *Dst) EnableReSend() {
	d.cbWg.Wait()
	d.reSend = true
}

// EnableRandDelay enables randomly delaying the first packet up to interval.
func (d *Dst) EnableRandDelay() {
	d.cbWg.Wait()
	d.randDelay = true
}

// EnableCallBackOnSend enables firing callback after each packet is sent.
// Packets sent but not yet recieved can be identified by by checking Recieved()
func (d *Dst) EnableCallBackOnSend() {
	d.cbWg.Wait()
	d.callBackOnSend = true
}

// Stop stops a runnning ping.
// The caller should wait until Run() returns after calling Stop().
func (d *Dst) Stop() {
	select {
	case _, open := <-d.stop:
		if !open {
			return
		}
	default:
	}
	close(d.stop)
}
