package messages

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/TrilliumIT/go-multiping/ping"
)

type Props struct {
	Network     string
	Src         string
	SendType    icmp.Type
	RecvType    icmp.Type
	ExpectedLen int
}

var V4Props = &Props{
	Network:  "ip4:icmp",
	Src:      "0.0.0.0",
	SendType: ipv4.ICMPTypeEcho,
	RecvType: ipv4.ICMPTypeEcho,
}

var V6Props = &Props{
	Network:  "ip6:ipv6-icmp",
	Src:      "::",
	SendType: ipv6.ICMPTypeEchoRequest,
	RecvType: ipv6.ICMPTypeEchoReply,
}

func init() {
	if m, err := (&icmp.Message{
		Type: V4Props.RecvType,
		Body: &icmp.Echo{
			Data: make([]byte, ping.TimeSliceLength),
		},
	}).Marshal(nil); err == nil {
		V4Props.ExpectedLen = len(m) + v4AddLen
	} else {
		panic(err)
	}

	if m, err := (&icmp.Message{
		Type: V6Props.RecvType,
		Body: &icmp.Echo{
			Data: make([]byte, ping.TimeSliceLength),
		},
	}).Marshal(nil); err == nil {
		V6Props.ExpectedLen = len(m) + v6AddLen
	} else {
		panic(err)
	}
}
