package ping

import (
	"fmt"
	"net"
	"sync"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// Pinger represents a pinger that can hold a raw-socket listener
type Pinger struct {
	v4Pinger *protoPinger
	v6Pinger *protoPinger
}

type protoPinger struct {
	stop      chan struct{}
	network   string
	src       string
	sendTyp   icmp.Type
	conn      *icmp.PacketConn
	cbLock    sync.RWMutex
	callbacks map[string]func(*Packet)
	wg        sync.WaitGroup
}

func NewPinger() *Pinger {
	return &Pinger{
		v4Pinger: &protoPinger{
			network: "ip4:icmp",
			src:     "0.0.0.0",
			sendTyp: ipv4.ICMPTypeEcho,
		},
		v6Pinger: &protoPinger{
			network: "ip6:ipv6-icmp",
			src:     "::",
			sendTyp: ipv6.ICMPTypeEchoRequest,
		},
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
