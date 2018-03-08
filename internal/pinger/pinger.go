package pinger

import (
	"fmt"
	"net"
	"sync"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/clinta/go-multiping/packet"
)

type Pinger struct {
	stop      chan struct{}
	network   string
	src       string
	sendType  icmp.Type
	Conn      *icmp.PacketConn
	cbLock    sync.RWMutex
	callbacks map[string]func(*packet.Packet)
	wg        sync.WaitGroup
}

func New(v int) *Pinger {
	p := &Pinger{
		callbacks: make(map[string]func(*packet.Packet)),
	}

	if v == 4 {
		p.network = "ip4:icmp"
		p.src = "0.0.0.0"
		p.sendType = ipv4.ICMPTypeEcho
	}

	if v == 6 {
		p.network = "ip6:ipv6-icmp"
		p.src = "::"
		p.sendType = ipv6.ICMPTypeEchoRequest
	}

	return p
}

func dstStr(ip net.IP, id int) string {
	return fmt.Sprintf("%s/%v", ip.String(), id)
}

func (pp *Pinger) SendType() icmp.Type {
	return pp.sendType
}

func (pp *Pinger) Network() string {
	return pp.network
}

func (pp *Pinger) GetCallback(ip net.IP, id int) (func(*packet.Packet), bool) {
	k := dstStr(ip, id)
	pp.cbLock.RLock()
	defer pp.cbLock.RUnlock()
	v, ok := pp.callbacks[k]
	return v, ok
}

func (pp *Pinger) AddCallBack(ip net.IP, id int, cb func(*packet.Packet)) error {
	if ip == nil {
		return fmt.Errorf("invalid ip")
	}
	if cb == nil {
		return fmt.Errorf("invalid callback")
	}
	k := dstStr(ip, id)
	pp.cbLock.Lock()
	defer pp.cbLock.Unlock()
	if _, ok := pp.callbacks[k]; ok {
		return fmt.Errorf("pinger %v already exists", k)
	}
	pp.callbacks[k] = cb
	if len(pp.callbacks) == 1 {
		err := pp.createListener()
		if err != nil {
			return err
		}
		pp.stop = make(chan struct{})
		pp.wg.Add(1)
		go func() {
			defer pp.wg.Done()
			pp.listen()
		}()
	}
	return nil
}

func (pp *Pinger) DelCallBack(ip net.IP, id int) {
	k := dstStr(ip, id)
	pp.cbLock.Lock()
	defer pp.cbLock.Unlock()
	delete(pp.callbacks, k)
	if len(pp.callbacks) == 0 {
		close(pp.stop)
		pp.wg.Wait()
	}
}
