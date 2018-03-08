package packet

import (
	"encoding/binary"
	"net"
	"time"
)

const (
	TimeSliceLength  = 8
	ProtocolICMP     = 1  // Internet Control Message
	ProtocolIPv6ICMP = 58 // ICMP for IPv6
)

type Packet struct {
	Src      net.IP
	Dst      net.IP
	ID       int
	TTL      int
	Len      int
	Seq      int
	Recieved time.Time
	Sent     time.Time
	RTT      time.Duration
}

func TimeToBytes(t time.Time) []byte {
	b := make([]byte, TimeSliceLength)
	binary.LittleEndian.PutUint64(b, uint64(t.UnixNano()))
	return b
}

func BytesToTime(b []byte) time.Time {
	return time.Unix(0, int64(binary.LittleEndian.Uint64(b[:TimeSliceLength])))
}
