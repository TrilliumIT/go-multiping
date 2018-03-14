package packet

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	// TimeSliceLength is the length of the icmp payload holding the timestamp
	TimeSliceLength = 8
)

// see https://godoc.org/golang.org/x/net/internal/iana#pkg-constants
const (
	ProtocolICMP     = 1
	ProtocolIPv6ICMP = 58 // ICMP for IPv6
)

// Packet is an ICMP packet that has been received
type Packet struct {
	// Src is the source IP. This is probably 0.0.0.0 for sent packets, but a
	// specific IP on the sending host for recieved packets
	Src net.IP
	// Dst is the destination IP
	Dst net.IP
	// ID is the ICMP ID
	ID int
	// Seq is the ICMP Sequence
	Seq int
	// Sent is the time the echo was sent
	Sent time.Time
	// Recieved is the time the echo was recieved.
	Recieved time.Time
	// TimedOut is the time the echo timed out
	TimedOut time.Time
	// TTL is the ttl on the recieved packet.
	TTL int
	// Len is the length of the recieved packet
	Len int
	// RTT is the round trip time of the packet
	RTT time.Duration
}

func (p *Packet) sendType() icmp.Type {
	if p.Dst.To4() != nil {
		return ipv4.ICMPTypeEcho
	}
	return ipv6.ICMPTypeEchoRequest
}

func (p *Packet) ToICMPMsg() ([]byte, error) {
	return (&icmp.Message{
		Code: 0,
		Type: p.sendType(),
		Body: &icmp.Echo{
			ID:   p.ID,
			Seq:  p.Seq,
			Data: TimeToBytes(p.Sent),
		},
	}).Marshal(nil)
}

// TimeToBytes converts a time.Time into a []byte for inclusion in the ICMP payload
func TimeToBytes(t time.Time) []byte {
	b := make([]byte, TimeSliceLength)
	binary.LittleEndian.PutUint64(b, uint64(t.UnixNano()))
	return b
}

// BytesToTime converst a []byte into a time.Time
func BytesToTime(b []byte) (time.Time, error) {
	if len(b) < TimeSliceLength {
		return time.Time{}, fmt.Errorf("too short")
	}
	return time.Unix(0, int64(binary.LittleEndian.Uint64(b[:TimeSliceLength]))), nil
}
