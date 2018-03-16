// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package pinger

import (
	"fmt"
	"net"
	"testing"

	"github.com/TrilliumIT/go-multiping/ping"
)

func TestOnSendError(t *testing.T) {
	var ips = []string{"0.0.0.1", "0.0.0.5"}
	cb := func(p *ping.Ping, err error, f func(j int)) {
		fmt.Printf("p: %v, err: %v\n", p, err)
		if err == nil {
			f(10)
			return
		}
		if _, ok := err.(*net.DNSError); !ok {
			f(1)
		} else {
			f(100)
		}
	}
	setup := func(d *Dst, f func(j int)) {
		d.EnableReSend()
	}
	testCallbacks(t, ips, 4, 0, setup, cb, 1)
}
