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

// Pinger is a protocol specific pinger, either ipv4 or ipv6
type Pinger struct {
	stop        chan struct{}
	network     string
	src         string
	sendType    icmp.Type
	Conn        *icmp.PacketConn
	cbLock      sync.RWMutex
	callbacks   map[[18]byte]func(*packet.Packet)
	expectedLen int
	closeWait   func() error
}

// New returns a new pinger
func New(v int) *Pinger {
	p := &Pinger{
		callbacks: make(map[[18]byte]func(*packet.Packet)),
	}

	var typ icmp.Type
	if v == 4 {
		p.network = "ip4:icmp"
		p.src = "0.0.0.0"
		p.sendType = ipv4.ICMPTypeEcho
		typ = ipv4.ICMPTypeEcho
	}

	if v == 6 {
		p.network = "ip6:ipv6-icmp"
		p.src = "::"
		p.sendType = ipv6.ICMPTypeEchoRequest
		typ = ipv6.ICMPTypeEchoReply
	}

	if m, err := (&icmp.Message{
		Type: typ,
		Body: &icmp.Echo{
			Data: make([]byte, packet.TimeSliceLength),
		},
	}).Marshal(nil); err == nil {
		p.expectedLen = len(m)
	} else {
		return nil
	}

	return p
}

func dstKey(ip net.IP, id uint16) [18]byte {
	var r [18]byte
	copy(r[0:16], ip.To16())
	binary.LittleEndian.PutUint16(r[16:], id)
	return r
}

// SendType returns the icmp.Type for the given protocol
func (pp *Pinger) SendType() icmp.Type {
	return pp.sendType
}

// Network returns the name of the network
func (pp *Pinger) Network() string {
	return pp.network
}

// GetCallback returns the OnRecieve callback for for a given IP and icmp id
func (pp *Pinger) GetCallback(ip net.IP, id uint16) (func(*packet.Packet), bool) {
	k := dstKey(ip, id)
	pp.cbLock.RLock()
	defer pp.cbLock.RUnlock()
	v, ok := pp.callbacks[k]
	return v, ok
}

// ErrorAlreadyExists is returned if a callback already exists.
// Used to detect icmp ID conflicts
type ErrorAlreadyExists struct{}

func (e *ErrorAlreadyExists) Error() string {
	return "callback already exists"
}

// AddCallBack adds an OnRecieve callback for a given IP and icmp id
// This implicitly starts the listening for these packets
func (pp *Pinger) AddCallBack(ip net.IP, id uint16, cb func(*packet.Packet)) error {
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
		return &ErrorAlreadyExists{}
	}
	pp.callbacks[k] = cb
	if len(pp.callbacks) == 1 {
		pp.stop = make(chan struct{})
		var err error
		pp.closeWait, err = pp.listen()
		if err != nil {
			return err
		}
	}
	return nil
}

// DelCallBack deletes a callback for a given IP and icmp id
// This stops listening for these packets
func (pp *Pinger) DelCallBack(ip net.IP, id uint16) error {
	k := dstKey(ip, id)
	pp.cbLock.Lock()
	defer pp.cbLock.Unlock()
	delete(pp.callbacks, k)
	var err error
	if len(pp.callbacks) == 0 {
		close(pp.stop)
		err = pp.closeWait()
	}
	return err
}
