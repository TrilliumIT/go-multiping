package packet

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
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

// SentPacket is an ICMP echo that has been sent, or attempted to be sent
type SentPacket struct {
	Dst  net.IP
	ID   int
	Seq  int
	Sent time.Time
}

// Packet is an ICMP packet that has been received
type Packet struct {
	SentPacket
	Src      net.IP
	TTL      int
	Len      int
	Recieved time.Time
	RTT      time.Duration
}

// ToSentPacket turns a packet into a SentPacket
func (p *Packet) ToSentPacket() *SentPacket {
	if p == nil {
		return nil
	}
	return &SentPacket{
		Dst:  p.Src,
		ID:   p.ID,
		Seq:  p.Seq,
		Sent: p.Sent,
	}
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
