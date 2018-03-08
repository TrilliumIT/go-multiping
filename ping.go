package ping

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sync"
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

type Packet struct {
	Src      net.IP
	Dst      net.IP
	ID       int
	TTL      int
	Recieved time.Time
	Sent     time.Time
	RTT      time.Duration
}

type recvMsg struct {
	v4cm       *ipv4.ControlMessage
	v6cm       *ipv6.ControlMessage
	src        net.Addr
	recieved   time.Time
	payload    []byte
	lenPayload int
}

type pingee struct {
	d net.Addr
	m *icmp.Message
	b *icmp.Echo
}

func dstStr(ip net.IP, id int) string {
	return fmt.Sprintf("%s/%v", ip.String(), id)
}

func ping(count int, interval time.Duration, stagger bool, onReply func(*Packet), dest ...string) (chan<- struct{}, error) {
	var wg sync.WaitGroup
	var listenV4, listenV6 bool
	var err error

	pingees := []*pingee{}
	dst := make(map[string]struct{})
	// dst/id
	for _, destStr := range dest {
		var typ icmp.Type
		dstAddr, err := net.ResolveIPAddr("ip", destStr)
		if err != nil {
			return nil, err
		}

		if dstAddr.IP == nil {
			return nil, fmt.Errorf("failed to get ip from %v", destStr)
		}

		if dstAddr.IP.To4() != nil {
			listenV4 = true
			typ = ipv4.ICMPTypeEcho
		} else {
			listenV6 = true
			typ = ipv6.ICMPTypeEchoRequest
		}
		id := rand.Intn(65535)
		dst[dstStr(dstAddr.IP, id)] = struct{}{}
		p := &pingee{
			d: dstAddr,
			b: &icmp.Echo{
				ID:  id,
				Seq: 0,
			},
			m: &icmp.Message{
				Type: typ,
				Code: 0,
			},
		}
		p.m.Body = p.b
		pingees = append(pingees, p)
	}

	stop := make(chan struct{})

	var v4Conn, v6Conn *icmp.PacketConn
	if listenV4 {
		network := "ip4:icmp"
		src := "0.0.0.0"
		v4Conn, err = icmp.ListenPacket(network, src)
		if err != nil {
			return stop, err
		}
		defer v4Conn.Close()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					r := &recvMsg{}
					b := []byte{}
					var err error
					r.lenPayload, r.v4cm, _, err = v4Conn.IPv4PacketConn().ReadFrom(b)
					if err != nil {
						//TODO something better here
						panic(err)
					}
					go processMessage(r, onReply, dst)
				}
			}
		}()
	}
	if listenV6 {
		network := "ip6:ipv6-icmp"
		src := "::"
		v6Conn, err = icmp.ListenPacket(network, src)
		if err != nil {
			return nil, err
		}
		defer v6Conn.Close()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					r := &recvMsg{}
					b := []byte{}
					var err error
					r.lenPayload, r.v6cm, _, err = v6Conn.IPv6PacketConn().ReadFrom(b)
					if err != nil {
						//TODO something better here
						panic(err)
					}
					go processMessage(r, onReply, dst)
				}
			}
		}()
	}

	var t *time.Ticker
	if stagger {
		t = time.NewTicker(time.Duration(interval.Nanoseconds() / int64(len(pingees))))
	} else {
		t = time.NewTicker(interval)
	}

	// immediate first run
	for ; true; <-t.C {
		for i, p := range pingees {
			if stagger && i != 0 {
				<-t.C
			}
			p.b.Data = timeToBytes(time.Now())
			b, err := p.m.Marshal(nil)
			if err != nil {
				continue
			}
			if p.m.Type == ipv4.ICMPTypeEcho {
				_, err = v4Conn.WriteTo(b, p.d)
			}
			if p.m.Type == ipv6.ICMPTypeEchoRequest {
				_, err = v4Conn.WriteTo(b, p.d)
			}
			if err != nil {
				continue
			}
			p.b.Seq += 1
		}
	}

	return nil, nil
}

func timeToBytes(t time.Time) []byte {
	b := make([]byte, timeSliceLength)
	binary.LittleEndian.PutUint64(b, uint64(t.UnixNano()))
	return b
}

func bytesToTime(b []byte) time.Time {
	return time.Unix(0, int64(binary.LittleEndian.Uint64(b)))
}
