package pinger

import (
	"context"
	"net"

	"github.com/TrilliumIT/go-multiping/internal/listenmap"
	"github.com/TrilliumIT/go-multiping/ping"
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

func addListener(ctx context.Context, lm *listenmap.ListenMap, ip net.IP, id uint16, h func(context.Context, *ping.Ping)) (func(), error) {
	lctx, lCancel := context.WithCancel(ctx)
	err := lm.Add(lctx, ip, id, h)
	return lCancel, err
}
