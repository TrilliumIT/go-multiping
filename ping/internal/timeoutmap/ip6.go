package timeoutmap

import (
	"encoding/binary"
	"net"
	"time"
)

func toIP6Idx(ip net.IP, id, seq int) [20]byte {
	var r [20]byte
	copy(r[0:16], ip.To16())
	binary.LittleEndian.PutUint16(r[16:18], uint16(id))
	binary.LittleEndian.PutUint16(r[18:], uint16(seq))
	return r
}

func fromIP6Idx(b [20]byte) (ip net.IP, id, seq int) {
	r := net.IP{}
	copy(r, b[0:16])
	return r,
		int(binary.LittleEndian.Uint16(b[16:18])),
		int(binary.LittleEndian.Uint16(b[18:]))
}

type ip6m map[[20]byte]time.Time

func (i ip6m) add(ip net.IP, id, seq int, t time.Time) {
	i[toIP6Idx(ip, id, seq)] = t
}

func (i ip6m) del(ip net.IP, id, seq int) {
	delete(i, toIP6Idx(ip, id, seq))
}

func (i ip6m) exists(ip net.IP, id, seq int) bool {
	_, ok := i[toIP6Idx(ip, id, seq)]
	return ok
}

func (i ip6m) getNext() (ip net.IP, id int, seq int, t time.Time) {
	for k, v := range i {
		if v.Before(t) || t.IsZero() {
			t = v
			ip, id, seq = fromIP6Idx(k)
		}
	}
	return
}
