package messages

import (
	"net"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/TrilliumIT/go-multiping/ping"
)

// Props holds various properties for an icmp message
type Props struct {
	Network     string
	Src         string
	SrcIP       net.IP
	SendType    icmp.Type
	RecvType    icmp.Type
	ExpectedLen int
}

// V4Props holds the properties for v4
var V4Props = &Props{
	Network:  "ip4:icmp",
	Src:      "0.0.0.0",
	SrcIP:    net.IPv4zero,
	SendType: ipv4.ICMPTypeEcho,
	RecvType: ipv4.ICMPTypeEcho,
}

// V6Props hods the properties for v6
var V6Props = &Props{
	Network:  "ip6:ipv6-icmp",
	Src:      "::",
	SrcIP:    net.IPv6zero,
	SendType: ipv6.ICMPTypeEchoRequest,
	RecvType: ipv6.ICMPTypeEchoReply,
}

func getLen(typ icmp.Type) int {
	m, err := (&icmp.Message{
		Type: typ,
		Body: &icmp.Echo{
			Data: make([]byte, ping.TimeSliceLength),
		},
	}).Marshal(nil)
	if err != nil {
		panic(err)
	}
	return len(m)
}

func init() {
	V4Props.ExpectedLen = getLen(V4Props.RecvType) + v4AddLen
	V6Props.ExpectedLen = getLen(V4Props.RecvType) + v6AddLen
}
