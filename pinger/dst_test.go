package pinger

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"

	"github.com/TrilliumIT/go-multiping/ping"
)

func TestMain(m *testing.M) {
	go func() {
		time.Sleep(60 * time.Second)
		_ = pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		panic("die")
	}()
	os.Exit(m.Run())
}

func checkGoRoutines(t *testing.T, igr int) {
	pr := pprof.Lookup("goroutine")
	gr := runtime.NumGoroutine()
	if gr > igr {
		_ = pr.WriteTo(os.Stdout, 1)
		t.Errorf("leaking goroutines: %v", gr-igr)
	}
}

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func testCallbacks(
	t *testing.T,
	ips []string,
	count int,
	setup func(d *Dst, f func(j int)),
	cb func(p *ping.Ping, e error, f func(j int)),
	countMultiplier int,
) {
	igr := runtime.NumGoroutine()
	ti := 0
	addTi := sync.Mutex{}
	wg := sync.WaitGroup{}
	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			i := 0
			f := func(j int) {
				addTi.Lock()
				defer addTi.Unlock()
				i += j
				ti += j
			}
			cbw := func(p *ping.Ping, err error) {
				if cb != nil {
					cb(p, err, f)
				}
			}
			d := NewDst(ip, 100*time.Millisecond, time.Second, count, cbw)
			if setup != nil {
				setup(d, f)
			}
			checkErr(t, d.Run())
			if i != count*countMultiplier {
				t.Errorf("only %v of %v packets counted for %v", i, count, ip)
			}
		}(ip)
	}
	wg.Wait()
	if ti != count*len(ips)*countMultiplier {
		t.Errorf("only %v of %v total packets counted", ti, count*len(ips))
	}
	time.Sleep(time.Millisecond)
	checkGoRoutines(t, igr)
}

func TestNoCallbacks(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	setup := func(d *Dst, f func(int)) {}
	testCallbacks(t, ips, 4, setup, nil, 0)
}

func TestOnReply(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	cb := func(p *ping.Ping, err error, f func(j int)) {
		if err == nil && p.IsRecieved() && !p.IsTimedOut() {
			f(1)
		}
	}
	testCallbacks(t, ips, 4, nil, cb, 1)
}

func TestOnTimeout(t *testing.T) {
	var ips = []string{"192.0.2.0", "198.51.100.0", "203.0.113.0", "fe80::2", "fe80::3", "fe80::4"}
	cb := func(p *ping.Ping, err error, f func(j int)) {
		if err == nil && p.IsTimedOut() {
			f(1)
		} else {
			fmt.Printf("p: %#v, err: %v", p, err)
			f(100)
		}
	}
	testCallbacks(t, ips, 4, nil, cb, 1)
}

func TestOnResolveError(t *testing.T) {
	var ips = []string{"foo.test", "bar.test", "baz.test"}
	cb := func(p *ping.Ping, err error, f func(j int)) {
		if _, ok := err.(*net.DNSError); ok {
			f(1)
		} else {
			f(100)
		}
	}
	setup := func(d *Dst, f func(j int)) {
		d.EnableReResolve()
		fmt.Printf("Reresolve enabled: %v\n", d.reResolve)
	}
	testCallbacks(t, ips, 4, setup, cb, 1)
}

func MultiValid(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	cb := func(p *ping.Ping, err error, f func(j int)) {
		if err == nil && p.IsRecieved() && !p.IsTimedOut() {
			f(1)
		} else {
			f(100)
		}
	}
	setup := func(d *Dst, f func(j int)) {
		d.EnableCallBackOnSend()
		d.EnableRandDelay()
		d.EnableReResolve()
		d.EnableReSend()
	}
	testCallbacks(t, ips, 4, setup, cb, 2)
}

// nolint:dupl
func MultiResolveError(t *testing.T) {
	var ips = []string{"foo.test", "bar.test", "baz.test"}
	cb := func(p *ping.Ping, err error, f func(j int)) {
		if _, ok := err.(*net.DNSError); ok {
			f(1)
		} else {
			f(100)
		}
	}
	setup := func(d *Dst, f func(j int)) {
		d.EnableCallBackOnSend()
		d.EnableRandDelay()
		d.EnableReResolve()
		d.EnableReSend()
	}
	testCallbacks(t, ips, 4, setup, cb, 2)
}

// nolint:dupl
func MultiSendError(t *testing.T) {
	var ips = []string{"0.0.0.1", "0.0.0.5"}
	cb := func(p *ping.Ping, err error, f func(j int)) {
		if _, ok := err.(*net.DNSError); !ok && err != nil {
			f(1)
		} else {
			f(100)
		}
	}
	setup := func(d *Dst, f func(j int)) {
		d.EnableCallBackOnSend()
		d.EnableRandDelay()
		d.EnableReResolve()
		d.EnableReSend()
	}
	testCallbacks(t, ips, 4, setup, cb, 2)
}
