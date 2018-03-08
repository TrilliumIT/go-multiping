package pinger

import (
	"fmt"
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
	// Count is the count of pings
	count int
	// OnReply is the callback triggered every time a ping is recieved
	onReply func(*packet.Packet)
	pinger  *protoPinger.Pinger
	stop    chan struct{}
}

// NewPingDest creates a PingDest object
func (p *Pinger) NewDst(dst string, interval, timeout time.Duration, count int, onReply func(*packet.Packet)) (*Dst, error) {
	d := &Dst{
		dstStr:   dst,
		interval: time.Second,
		timeout:  time.Second,
		count:    count,
		onReply:  onReply,
		stop:     make(chan struct{}),
	}

	var err error
	d.dst, err = net.ResolveIPAddr("ip", dst)
	fmt.Printf("%p\n", d.dst)
	if err != nil {
		return nil, err
	}
	//d.dst, _ = net.ResolveIPAddr("ip", d.dst.IP.String())

	d.pinger = p.getProtoPinger(d.dst.IP)
	return d, err
}

func (d *Dst) Stop() {
	close(d.stop)
}
