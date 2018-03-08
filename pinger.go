package ping

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// PingDest is a ping destination
type PingDest struct {
	// Dest is the net.IP to be pinged
	dest net.IP
	// Interval is the ping interval
	interval time.Duration
	// Timeout is how long to wait after the last packet is sent
	timeout time.Duration
	// Count is the count of pings
	count int
	// OnReply is the callback triggered every time a ping is recieved
	onReply func(*Packet)
	pinger  *Pinger
	stop    chan struct{}
}

type Packet struct {
	Src      net.IP
	Dst      net.IP
	ID       int
	TTL      int
	Recieved time.Time
	Sent     time.Time
	RTT      time.Duration
}

// Pinger represents a pinger that can hold a raw-socket listener
type Pinger struct {
	v4Pinger *protoPinger
	v6Pinger *protoPinger
}

type protoPinger struct {
	stop      chan struct{}
	network   string
	src       string
	cbLock    sync.RWMutex
	callbacks map[string]func(*Packet)
	wg        sync.WaitGroup
}

func NewPinger() *Pinger {
	return &Pinger{
		v4Pinger: &protoPinger{network: "ip4:icmp", src: "0.0.0.0"},
		v6Pinger: &protoPinger{network: "ip6:ipv6-icmp", src: "::"},
	}
}

func dstStr(ip net.IP, id int) string {
	return fmt.Sprintf("%s/%v", ip.String(), id)
}

func (p *Pinger) getProtoPinger(ip net.IP) *protoPinger {
	if ip != nil && ip.To4() != nil {
		return p.v4Pinger
	}
	return p.v6Pinger
}

func (p *Pinger) getCallback(ip net.IP, id int) (func(*Packet), bool) {
	pp := p.getProtoPinger(ip)
	return pp.getCallback(ip, id)
}

func (pp *protoPinger) getCallback(ip net.IP, id int) (func(*Packet), bool) {
	k := dstStr(ip, id)
	pp.cbLock.RLock()
	defer pp.cbLock.RUnlock()
	v, ok := pp.callbacks[k]
	return v, ok
}

func (p *Pinger) addCallBack(ip net.IP, id int, cb func(*Packet)) error {
	pp := p.getProtoPinger(ip)
	return pp.addCallBack(ip, id, cb)
}

func (pp *protoPinger) addCallBack(ip net.IP, id int, cb func(*Packet)) error {
	k := dstStr(ip, id)
	pp.cbLock.Lock()
	defer pp.cbLock.Unlock()
	if _, ok := pp.callbacks[k]; ok {
		return fmt.Errorf("pinger %v already exists", k)
	}
	pp.callbacks[k] = cb
	if len(pp.callbacks) == 1 {
		pp.wg.Add(1)
		go func() {
			defer pp.wg.Done()
			pp.listen()
		}()
	}
	return nil
}

func (p *Pinger) delCallBack(ip net.IP, id int) {
	pp := p.getProtoPinger(ip)
	pp.delCallBack(ip, id)
}

func (pp *protoPinger) delCallBack(ip net.IP, id int) {
	k := dstStr(ip, id)
	pp.cbLock.Lock()
	defer pp.cbLock.Unlock()
	delete(pp.callbacks, k)
	if len(pp.callbacks) == 0 {
		close(pp.stop)
		pp.wg.Wait()
	}
}

// NewPingDest creates a PingDest object
func (p *Pinger) NewPingDest(dest net.IP, interval, timeout time.Duration, count int, onReply func(*Packet)) *PingDest {
	return &PingDest{
		dest:     dest,
		interval: time.Second,
		timeout:  time.Second,
		count:    count,
		onReply:  onReply,
		pinger:   p,
		stop:     make(chan struct{}),
	}
}
