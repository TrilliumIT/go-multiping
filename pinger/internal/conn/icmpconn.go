package conn

import (
	"errors"
	"net"
	"time"

	"golang.org/x/net/icmp"

	"github.com/TrilliumIT/go-multiping/ping"
)

func newConn(proto int) conn {
	switch proto {
	case 4:
		return &v4Conn{}
	case 6:
		return &v6Conn{}
	default:
		panic("bad protocol")
	}
}

type icmpConn struct {
	c *icmp.PacketConn
}

func (c *icmpConn) writeTo(b []byte, dst *net.IPAddr) (int, error) {
	return c.c.WriteTo(b, dst)
}

func (c *icmpConn) close() error {
	return c.c.Close()
}

func toPing(
	srcAddr net.Addr,
	src, dst net.IP,
	rlen int,
	ttl int,
	received time.Time,
) *ping.Ping {
	p := &ping.Ping{
		// Src and Dst are flipped, cause this is a ping that was sent to dst
		// which has now come back
		Src:      &net.IPAddr{IP: dst},
		Dst:      &net.IPAddr{IP: src},
		Len:      rlen,
		TTL:      ttl,
		Recieved: received,
	}
	if srcIPAddr, ok := srcAddr.(*net.IPAddr); ok {
		if srcIPAddr.IP != nil {
			p.Src = srcIPAddr
		}
	}

	return p
}

var ErrTooShort = errors.New("too short")
var ErrWrongType = errors.New("wrong type")
var ErrNotEcho = errors.New("not echo")

func parseEcho(
	proto int,
	typ icmp.Type,
	payload []byte,
	rlen int,
) (
	id, seq int,
	sent time.Time,
	err error,
) {
	if len(payload) < rlen {
		err = ErrTooShort
		return
	}

	var m *icmp.Message
	m, err = icmp.ParseMessage(proto, payload[:rlen])
	if err != nil {
		return
	}
	if m.Type != typ {
		err = ErrWrongType
		return
	}

	e, ok := m.Body.(*icmp.Echo)
	if !ok {
		err = ErrNotEcho
		return
	}

	id = e.ID
	seq = e.Seq

	sent, err = ping.BytesToTime(e.Data)
	return
}
