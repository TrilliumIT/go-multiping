package pinger

import (
	"time"

	"github.com/clinta/go-multiping/packet"
)

// PingDest is a ping destination
type Dst struct {
	host string
	// Interval is the ping interval
	interval time.Duration
	// Timeout is how long to wait after the last packet is sent
	timeout time.Duration
	// Count is the count of pings to be sent
	count int

	// callbacks
	onReply        func(*packet.Packet)
	onSend         func(*packet.SentPacket)
	onSendError    func(*packet.SentPacket, error)
	onTimeout      func(*packet.SentPacket)
	onResolveError func(*packet.SentPacket, error)
	//onOutOfOrder func(*packet.Packet)

	pinger *Pinger
	stop   chan struct{}
}

// NewDst creates a PingDest object
func NewDst(host string, interval, timeout time.Duration, count int) *Dst {
	return getGlobalPinger().NewDst(host, interval, timeout, count)
}

// NewDst creates a PingDest object
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

func (d *Dst) SetOnReply(f func(*packet.Packet)) {
	d.onReply = f
}

func (d *Dst) SetOnSend(f func(*packet.SentPacket)) {
	d.onSend = f
}

func (d *Dst) SetOnSendError(f func(*packet.SentPacket, error)) {
	d.onSendError = f
}

func (d *Dst) SetOnTimeout(f func(*packet.SentPacket)) {
	d.onTimeout = f
}

// SetOnResolveError sets a callback to be called when a resolution
// error occurs. If this is not set, the host is only resolved once.
// If this is set, the host is re-resolved before sending each ping.
func (d *Dst) SetOnResolveError(f func(*packet.SentPacket, error)) {
	d.onResolveError = f
}

/*
func (d *Dst) SetOnOutOfOrder(f func(*packet.Packet)) {
	d.onOutOfOrder = f
}
*/

func (d *Dst) Stop() {
	close(d.stop)
}
