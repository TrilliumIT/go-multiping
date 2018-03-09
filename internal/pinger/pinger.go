package pinger

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/clinta/go-multiping/packet"
)

type Pinger struct {
	stop        chan struct{}
	network     string
	src         string
	sendType    icmp.Type
	Conn        *icmp.PacketConn
	cbLock      sync.RWMutex
	callbacks   map[[18]byte]func(*packet.Packet)
	wg          sync.WaitGroup
	expectedLen int
}

func New(v int) *Pinger {
	p := &Pinger{
		callbacks: make(map[[18]byte]func(*packet.Packet)),
	}

	if v == 4 {
		p.network = "ip4:icmp"
		p.src = "0.0.0.0"
		p.sendType = ipv4.ICMPTypeEcho

		if m, err := (&icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Body: &icmp.Echo{
				Data: make([]byte, packet.TimeSliceLength, packet.TimeSliceLength),
			},
		}).Marshal(nil); err == nil {
			p.expectedLen = len(m)
		} else {
			panic(err)
		}
	}

	if v == 6 {
		p.network = "ip6:ipv6-icmp"
		p.src = "::"
		p.sendType = ipv6.ICMPTypeEchoRequest
		if m, err := (&icmp.Message{
			Type: ipv6.ICMPTypeEchoReply,
			Body: &icmp.Echo{
				Data: make([]byte, packet.TimeSliceLength, packet.TimeSliceLength),
			},
		}).Marshal(nil); err == nil {
			p.expectedLen = len(m)
		} else {
			panic(err)
		}
	}

	return p
}

func dstKey(ip net.IP, id int) [18]byte {
	var r [18]byte
	copy(r[0:15], ip.To16())
	binary.LittleEndian.PutUint16(r[16:], uint16(id))
	return r
}

func (pp *Pinger) SendType() icmp.Type {
	return pp.sendType
}

func (pp *Pinger) Network() string {
	return pp.network
}

func (pp *Pinger) GetCallback(ip net.IP, id int) (func(*packet.Packet), bool) {
	k := dstKey(ip, id)
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
	k := dstKey(ip, id)
	pp.cbLock.Lock()
	defer pp.cbLock.Unlock()
	if _, ok := pp.callbacks[k]; ok {
		return fmt.Errorf("pinger %v already exists", k)
	}
	pp.callbacks[k] = cb
	if len(pp.callbacks) == 1 {
		pp.stop = make(chan struct{})
		wait, err := pp.listen()
		pp.wg.Add(1)
		go func() {
			wait()
			pp.wg.Done()
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func (pp *Pinger) DelCallBack(ip net.IP, id int) {
	k := dstKey(ip, id)
	pp.cbLock.Lock()
	defer pp.cbLock.Unlock()
	delete(pp.callbacks, k)
	if len(pp.callbacks) == 0 {
		close(pp.stop)
		pp.wg.Wait()
	}
}
