package pinger

import (
	"sync"
	"time"

	"github.com/clinta/go-multiping/packet"
)

// Dst is a destination host to be pinged
type Dst struct {
	host string
	// Interval is the ping interval
	interval time.Duration
	// Timeout is how long to wait after the last packet is sent
	timeout time.Duration
	// Count is the count of pings to be sent
	count int

	// randDelay controls whether the first ping will be dalayed by a random amount of time between 0 and interval
	randDelay bool
	// callbacks
	onReply        func(*packet.Packet)
	onSend         func(*packet.Packet)
	onSendError    func(*packet.Packet, error)
	onTimeout      func(*packet.Packet)
	onResolveError func(*packet.Packet, error)
	//onOutOfOrder func(*packet.Packet)

	pinger  *Pinger
	stop    chan struct{}
	sending chan struct{}
	cbWg    sync.WaitGroup
	pktCh   chan *pkt
}

// NewDst creates a Dst
func NewDst(host string, interval, timeout time.Duration, count int) *Dst {
	return getGlobalPinger().NewDst(host, interval, timeout, count)
}

// NewDst creates a Dst
func (p *Pinger) NewDst(host string, interval, timeout time.Duration, count int) *Dst {
	return &Dst{
		host:     host,
		interval: interval,
		timeout:  timeout,
		count:    count,
		pinger:   p,
		stop:     make(chan struct{}),
	}
}

// SetOnReply sets f to be called every time an ICMP reply is recieved
func (d *Dst) SetOnReply(f func(*packet.Packet)) {
	d.cbWg.Wait()
	d.onReply = f
}

// SetOnSend sets f to be called every time a packet is about to be sent
func (d *Dst) SetOnSend(f func(*packet.Packet)) {
	d.cbWg.Wait()
	d.onSend = f
}

// SetOnSendError sets f to be called every time an error is encountered sending. For example a no-route to host error.
// If this is not set, Run() will stop and return error when sending encounters an error
func (d *Dst) SetOnSendError(f func(*packet.Packet, error)) {
	d.cbWg.Wait()
	d.onSendError = f
}

// SetOnTimeout sets f to be called every time an ICMP reply is not recieved within timeout
func (d *Dst) SetOnTimeout(f func(*packet.Packet)) {
	d.cbWg.Wait()
	d.onTimeout = f
}

// SetOnResolveError sets a callback to be called when a resolution
// error occurs. If this is not set, the host is only resolved once at the beginning of Run(). If an error occurs at this time, it is returned to Run().
// If this is set, the host is re-resolved before sending each ping.
func (d *Dst) SetOnResolveError(f func(*packet.Packet, error)) {
	d.cbWg.Wait()
	d.onResolveError = f
}

// EnableRandDelay enables randomly delaying the first packet up to interval.
func (d *Dst) EnableRandDelay() {
	d.cbWg.Wait()
	d.randDelay = true
}

// Stop stops a runnning ping. This will panic if the ping is not running.
// The caller should wait until Run() returns after calling Stop().
func (d *Dst) Stop() {
	close(d.stop)
}
