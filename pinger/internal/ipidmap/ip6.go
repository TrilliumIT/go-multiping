package socket

import (
	"encoding/binary"
	"net"

	"github.com/TrilliumIT/go-multiping/pinger/internal/seqmap"
)

func toIP6Idx(ip net.IP, id int) [18]byte {
	var r [18]byte
	copy(r[0:16], ip.To16())
	binary.LittleEndian.PutUint16(r[16:], uint16(id))
	return r
}

type ip6IdMap map[[18]byte]*seqmap.SeqMap

func (i ip6IdMap) add(ip net.IP, id int, sm *seqmap.SeqMap) {
	i[toIP6Idx(ip, id)] = sm
}

func (i ip6IdMap) del(ip net.IP, id int) {
	delete(i, toIP6Idx(ip, id))
}

func (i ip6IdMap) get(ip net.IP, id int) (*seqmap.SeqMap, bool) {
	sm, ok := i[toIP6Idx(ip, id)]
	return sm, ok
}

func (i ip6IdMap) length() int {
	return len(i)
}
