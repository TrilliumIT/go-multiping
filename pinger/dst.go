package pinger

import (
	"net"
	"time"

	protoPinger "github.com/clinta/go-multiping/internal/pinger"
	"github.com/clinta/go-multiping/packet"
)

// PingDest is a ping destination
type Dst struct {
	// Dest is the net.IP to be pinged
	dst    *net.IPAddr
	dstStr string
	// Interval is the ping interval
	interval time.Duration
	// Timeout is how long to wait after the last packet is sent
	timeout time.Duration
	// Count is the count of pings to be sent
	count int

	// callbacks
	onReply     func(*packet.Packet)
	onSend      func(*packet.SentPacket)
	onSendError func(*packet.SentPacket)
	onTimeout   func(*packet.SentPacket)
	//onOutOfOrder func(*packet.Packet)
	// expectedLen is the expected lenght of incoming packets
	expectedLen int
	pinger      *protoPinger.Pinger
	stop        chan struct{}
}

// NewDst creates a PingDest object
func NewDst(dst string, interval, timeout time.Duration, count int) (*Dst, error) {
	return getGlobalPinger().NewDst(dst, interval, timeout, count)
}

// NewDst creates a PingDest object
func (p *Pinger) NewDst(dst string, interval, timeout time.Duration, count int) (*Dst, error) {
	d := &Dst{
		dstStr:   dst,
		interval: interval,
		timeout:  timeout,
		count:    count,
		stop:     make(chan struct{}),
	}

	var err error
	d.dst, err = net.ResolveIPAddr("ip", dst)
	if err != nil {
		return nil, err
	}
	//d.dst, _ = net.ResolveIPAddr("ip", d.dst.IP.String())

	d.pinger = p.getProtoPinger(d.dst.IP)
	return d, err
}

func (d *Dst) SetOnReply(f func(*packet.Packet)) {
	d.onReply = f
}

func (d *Dst) SetOnSend(f func(*packet.SentPacket)) {
	d.onSend = f
}

func (d *Dst) SetOnSendError(f func(*packet.SentPacket)) {
	d.onSendError = f
}

func (d *Dst) SetOnTimeout(f func(*packet.SentPacket)) {
	d.onTimeout = f
}

/*
func (d *Dst) SetOnOutOfOrder(f func(*packet.Packet)) {
	d.onOutOfOrder = f
}
*/

func (d *Dst) Stop() {
	close(d.stop)
}
