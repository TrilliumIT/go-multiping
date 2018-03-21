package socket

import (
	"encoding/binary"
	"net"

	"github.com/TrilliumIT/go-multiping/pinger/internal/seqmap"
)

func toIP4Idx(ip net.IP, id int) [6]byte {
	var r [6]byte
	copy(r[0:4], ip.To4())
	binary.LittleEndian.PutUint16(r[4:], uint16(id))
	return r
}

type ip4IdMap map[[6]byte]*seqmap.SeqMap

func (i ip4IdMap) add(ip net.IP, id int, sm *seqmap.SeqMap) {
	i[toIP4Idx(ip, id)] = sm
}

func (i ip4IdMap) del(ip net.IP, id int) {
	delete(i, toIP4Idx(ip, id))
}

func (i ip4IdMap) get(ip net.IP, id int) (*seqmap.SeqMap, bool) {
	sm, ok := i[toIP4Idx(ip, id)]
	return sm, ok
}

func (i ip4IdMap) length() int {
	return len(i)
}
