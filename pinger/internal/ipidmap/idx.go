package socket

import (
	"encoding/binary"
	"net"
)

func toIP4Idx(ip net.IP, id int) [6]byte {
	var r [6]byte
	copy(r[0:4], ip.To4())
	binary.LittleEndian.PutUint16(r[4:], uint16(id))
	return r
}

func toIP6Idx(ip net.IP, id int) [18]byte {
	var r [18]byte
	copy(r[0:16], ip.To16())
	binary.LittleEndian.PutUint16(r[16:], uint16(id))
	return r
}
