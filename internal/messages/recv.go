package messages

import (
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/TrilliumIT/go-multiping/ping"
)

type RecvMsg struct {
	V4cm       *ipv4.ControlMessage
	V6cm       *ipv6.ControlMessage
	Recieved   time.Time
	Payload    []byte
	PayloadLen int
}

func (r *RecvMsg) Props() (src, dst net.IP, ttl int, proto int, typ icmp.Type) {
	switch {
	case r.V4cm != nil:
		dst = r.V4cm.Src
		src = r.V4cm.Dst
		ttl = r.V4cm.TTL
		proto = ping.ProtocolICMP
		typ = ipv4.ICMPTypeEchoReply
	case r.V6cm != nil:
		dst = r.V6cm.Src
		src = r.V6cm.Dst
		ttl = r.V6cm.HopLimit
		proto = ping.ProtocolIPv6ICMP
		typ = ipv6.ICMPTypeEchoReply
	}
	return
}
