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
	dst net.IP
	// Interval is the ping interval
	interval time.Duration
	// Timeout is how long to wait after the last packet is sent
	timeout time.Duration
	// Count is the count of pings
	count int
	// OnReply is the callback triggered every time a ping is recieved
	onReply func(*packet.Packet)
	pinger  *protoPinger.Pinger
	stop    chan struct{}
}

// NewPingDest creates a PingDest object
func (p *Pinger) NewDst(dst net.IP, interval, timeout time.Duration, count int, onReply func(*packet.Packet)) *Dst {
	pp := p.getProtoPinger(dst)
	return &Dst{
		dst:      dst,
		interval: time.Second,
		timeout:  time.Second,
		count:    count,
		onReply:  onReply,
		pinger:   pp,
		stop:     make(chan struct{}),
	}
}

func (d *Dst) Stop() {
	close(d.stop)
}
