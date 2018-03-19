package ping

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

// Ping is an ICMP packet that has been received
type Ping struct {
	// Host is the hostname that was pinged
	Host string
	// Src is the source IP. This is probably 0.0.0.0 for sent packets, but a
	// specific IP on the sending host for recieved packets
	Src net.IP
	// Dst is the destination IP.
	// This will be nil for recieved packets on windows. The reason is that
	// the recieve function does not provide the source address
	// on windows ICMP messages are mathed only by the 16 bit ICMP id.
	Dst net.IP
	// ID is the ICMP ID
	ID int
	// Seq is the ICMP Sequence
	Seq int
	// Sent is the time the echo was sent
	Sent time.Time
	// Recieved is the time the echo was recieved.
	Recieved time.Time
	// TimeOut is timeout duration
	TimeOut time.Duration
	// TTL is the ttl on the recieved packet.
	// This is not supported on windows and will always be zero
	TTL int
	// Len is the length of the recieved packet
	Len int
}

// UpdateFrom is for udpdating a sent ping with attributes from a recieved ping
func (p *Ping) UpdateFrom(p2 *Ping) {
	if p.Host == "" {
		p.Host = p2.Host
	}

	if p.Src == nil && p2.Src != nil {
		p.Src = p2.Src
	}

	if p.Src.IsUnspecified() && !p2.Src.IsUnspecified() {
		p.Src = p2.Src
	}

	if p.Dst == nil && p2.Dst != nil {
		p.Dst = p2.Dst
	}

	if p.Dst.IsUnspecified() && !p2.Dst.IsUnspecified() {
		p.Dst = p2.Dst
	}

	if p.ID == 0 {
		p.ID = p2.ID
	}

	if p.Seq == 0 {
		p.Seq = p2.Seq
	}

	if p.Sent.IsZero() {
		p.Sent = p2.Sent
	}

	if p.Recieved.IsZero() {
		p.Recieved = p2.Recieved
	}

	if p.TimeOut == 0 {
		p.TimeOut = p2.TimeOut
	}

	if p.Len == 0 {
		p.Len = p2.Len
	}

	if p.TTL == 0 {
		p.TTL = p2.TTL
	}
}

// RTT returns the RTT of the ping
func (p *Ping) RTT() time.Duration {
	if !p.Recieved.Before(p.Sent) {
		return p.Recieved.Sub(p.Sent)
	}
	return 0
}

func (p *Ping) sendType() icmp.Type {
	if p.Dst.To4() != nil {
		return ipv4.ICMPTypeEcho
	}
	return ipv6.ICMPTypeEchoRequest
}

// ToICMPMsg returns a byte array ready to send on the wire
func (p *Ping) ToICMPMsg() ([]byte, error) {
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
