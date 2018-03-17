package pinger

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/TrilliumIT/go-multiping/ping"
)

type props struct {
	network     string
	src         string
	sendType    icmp.Type
	recvType    icmp.Type
	expectedLen int
}

var v4Props = &props{
	network:  "ip4:icmp",
	src:      "0.0.0.0",
	sendType: ipv4.ICMPTypeEcho,
	recvType: ipv4.ICMPTypeEcho,
}

var v6Props = &props{
	network:  "ip6:ipv6-icmp",
	src:      "::",
	sendType: ipv6.ICMPTypeEchoRequest,
	recvType: ipv6.ICMPTypeEchoReply,
}

func init() {
	if m, err := (&icmp.Message{
		Type: v4Props.recvType,
		Body: &icmp.Echo{
			Data: make([]byte, ping.TimeSliceLength),
		},
	}).Marshal(nil); err == nil {
		v4Props.expectedLen = len(m) + v4AddLen
	} else {
		panic(err)
	}

	if m, err := (&icmp.Message{
		Type: v6Props.recvType,
		Body: &icmp.Echo{
			Data: make([]byte, ping.TimeSliceLength),
		},
	}).Marshal(nil); err == nil {
		v6Props.expectedLen = len(m) + v6AddLen
	} else {
		panic(err)
	}
}
