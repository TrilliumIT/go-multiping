package pinger

import (
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/clinta/go-multiping/packet"
)

type recvMsg struct {
	v4cm       *ipv4.ControlMessage
	v6cm       *ipv6.ControlMessage
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
		p.Src = r.v4cm.Src
		p.Dst = r.v4cm.Dst
		p.TTL = uint8(r.v4cm.TTL)
		proto = packet.ProtocolICMP
		typ = ipv4.ICMPTypeEchoReply
	}
	if r.v6cm != nil {
		p.Src = r.v6cm.Src
		p.Dst = r.v6cm.Dst
		p.TTL = uint8(r.v6cm.HopLimit)
		proto = packet.ProtocolIPv6ICMP
		typ = ipv6.ICMPTypeEchoReply
	}
	if p.Dst == nil {
		return
	}

	if len(r.payload) < r.payloadLen {
		return
	}

	var m *icmp.Message
	var err error
	m, err = icmp.ParseMessage(proto, r.payload[:r.payloadLen])
	if err != nil {
		return
	}
	if m.Type != typ {
		return
	}

	e, ok := m.Body.(*icmp.Echo)
	if !ok {
		return
	}
	p.ID = uint16(e.ID)
	p.Seq = uint16(e.Seq)

	p.Sent, err = packet.BytesToTime(e.Data)
	if err != nil {
		return
	}

	cb, ok := pp.GetCallback(p.Src, p.ID)
	if !ok {
		return
	}

	if cb == nil {
		return
	}

	p.Len = r.payloadLen
	p.RTT = p.Recieved.Sub(p.Sent)

	cb(p)
}
