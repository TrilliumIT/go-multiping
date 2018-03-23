package ping

import (
	"net"
)

// returns true if dst is changed
func resolve(dst *net.IPAddr, host string) (*net.IPAddr, bool, error) {
	rDst := dst
	nDst, err := net.ResolveIPAddr("ip", host)
	changed := dst == nil || nDst == nil || !nDst.IP.Equal(dst.IP)
	if changed {
		rDst = nDst
	}
	return rDst, changed, err
}
