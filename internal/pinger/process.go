package pinger

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/clinta/go-multiping/packet"
)

type recvMsg struct {
	v4cm       *ipv4.ControlMessage
	v6cm       *ipv6.ControlMessage
	src        net.Addr
	recieved   time.Time
	payload    []byte
	payloadLen int
}

func (pp *Pinger) processMessage(r *recvMsg) {
	var proto int
	var typ icmp.Type
	p := &packet.Packet{}
	p.Recieved = r.recieved
	if r.v4cm != nil {
		fmt.Println("This is a v4 packet")
		p.Src = r.v4cm.Src
		p.Dst = r.v4cm.Dst
		p.TTL = r.v4cm.TTL
		proto = packet.ProtocolICMP
		typ = ipv4.ICMPTypeEchoReply
	}
	if r.v6cm != nil {
		fmt.Println("This is a v6 packet")
		p.Src = r.v6cm.Src
		p.Dst = r.v6cm.Dst
		p.TTL = r.v6cm.HopLimit
		proto = packet.ProtocolIPv6ICMP
		typ = ipv6.ICMPTypeEchoReply
	}
	if p.Dst == nil {
		fmt.Println("dst is nil")
		return
	}

	if len(r.payload) < r.payloadLen {
		fmt.Println("payload too short")
		return
	}

	var m *icmp.Message
	var err error
	m, err = icmp.ParseMessage(proto, r.payload[:r.payloadLen])
	if err != nil {
		fmt.Println("unable to parse")
		return
	}
	if m.Type != typ {
		return
	}

	e, ok := m.Body.(*icmp.Echo)
	if !ok {
		fmt.Println("not an echo")
		return
	}
	p.ID = e.ID

	if len(e.Data) < packet.TimeSliceLength {
		fmt.Println("echo data too short")
		return
	}

	cb, ok := pp.GetCallback(p.Src, p.ID)
	if !ok {
		fmt.Println("no callback")
		return
	}

	p.Len = r.payloadLen
	p.Sent = packet.BytesToTime(e.Data)
	p.RTT = p.Recieved.Sub(p.Sent)

	cb(p)
}
