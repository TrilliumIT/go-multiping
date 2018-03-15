package pinger

import (
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
			d := NewDst(ip, 100*time.Millisecond, time.Second, count)
			setup(d, f)
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
	testCallbacks(t, ips, 4, setup, 0)
}

func TestOnReply(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	setup := func(d *Dst, f func(j int)) {
		d.SetOnReply(func(*ping.Ping) { f(1) })
	}
	testCallbacks(t, ips, 4, setup, 1)
}

func TestOnTimeout(t *testing.T) {
	var ips = []string{"192.0.2.0", "198.51.100.0", "203.0.113.0", "fe80::2", "fe80::3", "fe80::4"}
	setup := func(d *Dst, f func(j int)) {
		d.SetOnTimeout(func(*ping.Ping) { f(1) })
	}
	testCallbacks(t, ips, 4, setup, 1)
}

func TestOnResolveError(t *testing.T) {
	var ips = []string{"foo.test", "bar.test", "baz.test"}
	setup := func(d *Dst, f func(j int)) {
		d.SetOnSendError(func(p *ping.Ping, err error) {
			if _, ok := err.(*net.DNSError); ok {
				f(1)
			}
		})
		d.EnableReResolve()
	}
	testCallbacks(t, ips, 4, setup, 1)
}

func MultiValid(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	setup := func(d *Dst, f func(j int)) {
		d.SetOnReply(func(p *ping.Ping) {
			f(1)
		})
		d.SetOnSend(func(*ping.Ping) {
			f(10)
		})
		d.SetOnSendError(func(*ping.Ping, error) {
			f(100)
		})
		d.EnableRandDelay()
	}
	testCallbacks(t, ips, 4, setup, 2)
}

// nolint:dupl
func MultiResolveError(t *testing.T) {
	var ips = []string{"foo.test", "bar.test", "baz.test"}
	setup := func(d *Dst, f func(j int)) {
		d.SetOnReply(func(p *ping.Ping) {
			f(100)
		})
		d.SetOnSend(func(*ping.Ping) {
			f(10)
		})
		d.SetOnSendError(func(p *ping.Ping, err error) {
			if _, ok := err.(*net.DNSError); ok {
				f(1)
				return
			}
			f(1000)
		})
		d.EnableRandDelay()
		d.EnableReResolve()
	}
	testCallbacks(t, ips, 4, setup, 1)
}

// nolint:dupl
func MultiSendError(t *testing.T) {
	var ips = []string{"0.0.0.1", "::2", "0.0.0.5", "::5"}
	setup := func(d *Dst, f func(j int)) {
		d.SetOnReply(func(p *ping.Ping) {
			f(10)
		})
		d.SetOnSend(func(*ping.Ping) {
			f(100)
		})
		d.SetOnSendError(func(p *ping.Ping, err error) {
			if _, ok := err.(*net.DNSError); ok {
				f(1000)
				return
			}
			f(1)
		})
		d.EnableReResolve()
		d.EnableRandDelay()
	}
	testCallbacks(t, ips, 4, setup, 1)
}
