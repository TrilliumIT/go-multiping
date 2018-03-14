package pinger

import (
	"github.com/clinta/go-multiping/packet"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	go func() {
		time.Sleep(60 * time.Second)
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		panic("die")
	}()
	os.Exit(m.Run())
}

func checkGoRoutines(t *testing.T, igr int) {
	pr := pprof.Lookup("goroutine")
	gr := runtime.NumGoroutine()
	if gr > igr {
		pr.WriteTo(os.Stdout, 1)
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
	setup func(d *Dst, f func()),
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
			f := func() {
				addTi.Lock()
				defer addTi.Unlock()
				i++
				ti++
			}
			d := NewDst(ip, 100*time.Millisecond, time.Second, count)
			setup(d, f)
			checkErr(t, d.Run())
			if i != count*countMultiplier {
				t.Errorf("only %v of %v packets counted", i, count)
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
	setup := func(d *Dst, f func()) {}
	testCallbacks(t, ips, 4, setup, 0)
}

func TestOnReply(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	setup := func(d *Dst, f func()) {
		d.SetOnReply(func(*packet.Packet) { f() })
	}
	testCallbacks(t, ips, 4, setup, 1)
}

func TestOnTimeout(t *testing.T) {
	var ips = []string{"192.0.2.0", "198.51.100.0", "203.0.113.0", "fe80::2", "fe80::3", "fe80::4"}
	setup := func(d *Dst, f func()) {
		d.SetOnTimeout(func(*packet.Packet) { f() })
	}
	testCallbacks(t, ips, 4, setup, 1)
}

func TestOnSendError(t *testing.T) {
	var ips = []string{"0.0.0.1", "::2", "0.0.0.5", "::5"}
	setup := func(d *Dst, f func()) {
		d.SetOnSendError(func(*packet.Packet, error) { f() })
	}
	testCallbacks(t, ips, 4, setup, 1)
}

func TestOnResolveError(t *testing.T) {
	var ips = []string{"foo.test", "bar.test", "baz.test"}
	setup := func(d *Dst, f func()) {
		d.SetOnResolveError(func(p *packet.Packet, err error) {
			f()
		})
	}
	testCallbacks(t, ips, 4, setup, 1)
}

func MultiValid(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	setup := func(d *Dst, f func()) {
		d.SetOnReply(func(p *packet.Packet) {
			f()
		})
		d.SetBeforeSend(func(*packet.Packet) {
			f()
		})
		d.SetOnSendError(func(*packet.Packet, error) {
			f()
		})
		d.SetOnResolveError(func(*packet.Packet, error) {
			f()
		})
		d.EnableRandDelay()
	}
	testCallbacks(t, ips, 4, setup, 2)
}

func MultiResolveError(t *testing.T) {
	var ips = []string{"foo.test", "bar.test", "baz.test"}
	setup := func(d *Dst, f func()) {
		d.SetOnReply(func(p *packet.Packet) {
			f()
		})
		d.SetBeforeSend(func(*packet.Packet) {
			f()
		})
		d.SetOnSendError(func(*packet.Packet, error) {
			f()
		})
		d.SetOnResolveError(func(*packet.Packet, error) {
			f()
		})
		d.EnableRandDelay()
	}
	testCallbacks(t, ips, 4, setup, 1)
}

func MultiSendError(t *testing.T) {
	var ips = []string{"0.0.0.1", "::2", "0.0.0.5", "::5"}
	setup := func(d *Dst, f func()) {
		d.SetOnReply(func(p *packet.Packet) {
			f()
		})
		d.SetBeforeSend(func(*packet.Packet) {
			f()
		})
		d.SetOnSendError(func(*packet.Packet, error) {
			f()
		})
		d.SetOnResolveError(func(*packet.Packet, error) {
			f()
		})
		d.EnableRandDelay()
	}
	testCallbacks(t, ips, 4, setup, 1)
}
