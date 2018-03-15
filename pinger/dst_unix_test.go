// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package pinger

import (
	"testing"

	"github.com/TrilliumIT/go-multiping/ping"
)

func TestOnSendError(t *testing.T) {
	var ips = []string{"0.0.0.1", "::2", "0.0.0.5", "::5"}
	setup := func(d *Dst, f func(j int)) {
		d.SetOnSendError(func(*ping.Ping, error) { f(1) })
	}
	testCallbacks(t, ips, 4, setup, 1)
}
