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
	// TimeOut is the time the echo timed out
	TimeOut time.Time
	// TTL is the ttl on the recieved packet.
	// This is not supported on windows and will always be zero
	TTL int
	// Len is the length of the recieved packet
	Len int

	// SendPening is open when a send is pending, but not yet succeeded
	SendPending chan struct{}
}

func (p *Ping) IsSending() bool {
	if p.SendPending == nil {
		return false
	}
	select {
	case <-p.SendPending:
		return false
	default:
	}
	return true
}

func (p *Ping) IsTimedOut() bool {
	if p.TimeOut.IsZero() {
		return false
	}
	if !p.Recieved.IsZero() {
		return p.Recieved.After(p.TimeOut)
	}
	return time.Now().After(p.TimeOut)
}

func (p *Ping) IsSent() bool {
	return !p.Sent.IsZero()
}

func (p *Ping) IsRecieved() bool {
	return !p.Recieved.IsZero()
}

func (p *Ping) IsPending() bool {
	return p.IsSent() && !p.IsRecieved() && !p.IsTimedOut()
}

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
