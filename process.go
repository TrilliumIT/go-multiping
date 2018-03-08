package ping

import (
	"encoding/binary"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	timeSliceLength  = 8
	ProtocolICMP     = 1  // Internet Control Message
	ProtocolIPv6ICMP = 58 // ICMP for IPv6
)

type recvMsg struct {
	v4cm       *ipv4.ControlMessage
	v6cm       *ipv6.ControlMessage
	src        net.Addr
	recieved   time.Time
	payload    []byte
	payloadLen int
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

func (pp *protoPinger) processMessage(r *recvMsg) {
	var proto int
	var typ icmp.Type
	p := &Packet{}
	p.Recieved = r.recieved
	if r.v4cm != nil {
		p.Src = r.v4cm.Src
		p.Dst = r.v4cm.Dst
		p.TTL = r.v4cm.TTL
		proto = ProtocolICMP
		typ = ipv4.ICMPTypeEchoReply
	}
	if r.v6cm != nil {
		p.Src = r.v6cm.Src
		p.Dst = r.v6cm.Dst
		p.TTL = r.v6cm.HopLimit
		proto = ProtocolIPv6ICMP
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
	p.ID = e.ID

	if len(e.Data) < timeSliceLength {
		return
	}

	cb, ok := pp.getCallback(p.Src, p.ID)
	if !ok {
		return
	}

	p.Sent = bytesToTime(e.Data[:timeSliceLength])
	p.RTT = p.Recieved.Sub(p.Sent)

	cb(p)
}

func bytesToTime(b []byte) time.Time {
	return time.Unix(0, int64(binary.LittleEndian.Uint64(b)))
}
