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

type cbTest struct {
	t               *testing.T
	ips             []string
	count           int
	interval        time.Duration
	timeout         time.Duration
	expire          time.Duration
	setup           func(d *Dst, f func(j int))
	cb              func(p *ping.Ping, e error, f func(j int))
	countMultiplier int
}

func testCallbacks(c *cbTest) {
	igr := runtime.NumGoroutine()
	ti := 0
	addTi := sync.Mutex{}
	wg := sync.WaitGroup{}
	for _, ip := range c.ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			i := 0
			f := func(j int) {
				addTi.Lock()
				i += j
				ti += j
				addTi.Unlock()
			}
			cbw := func(p *ping.Ping, err error) {
				if c.cb != nil {
					c.cb(p, err, f)
				}
			}
			d := NewDst(ip, c.interval, c.timeout, c.count, cbw)
			if c.setup != nil {
				c.setup(d, f)
			}
			if c.expire > 0 {
				go func() {
					time.Sleep(c.expire)
					d.Stop()
				}()
			}
			checkErr(c.t, d.Run())
			addTi.Lock()
			if i != c.count*c.countMultiplier {
				c.t.Errorf("only %v of %v packets counted for %v", i, c.count, ip)
			}
			addTi.Unlock()
		}(ip)
	}
	wg.Wait()
	if ti != c.count*len(c.ips)*c.countMultiplier {
		c.t.Errorf("only %v of %v total packets counted", ti, c.count*len(c.ips)*c.countMultiplier)
	}
	// this should not be necessary, but I got to figure it out
	time.Sleep(time.Millisecond)
	checkGoRoutines(c.t, igr)
}

func TestNoCallbacks(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	setup := func(d *Dst, f func(int)) {}
	testCallbacks(&cbTest{
		t:               t,
		ips:             ips,
		count:           4,
		timeout:         time.Second,
		interval:        time.Second,
		countMultiplier: 0,
		setup:           setup,
	})
}

func TestOnReply(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	cb := func(p *ping.Ping, err error, f func(j int)) {
		if err == nil && p.IsRecieved() && !p.IsTimedOut() {
			f(1)
		}
	}
	testCallbacks(&cbTest{
		t:               t,
		ips:             ips,
		count:           4,
		timeout:         time.Second,
		interval:        time.Second,
		countMultiplier: 1,
		cb:              cb,
	})
}

func TestOnReplyExpire(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	cb := func(p *ping.Ping, err error, f func(j int)) {
		if err == nil && p.IsRecieved() && !p.IsTimedOut() {
			f(0)
			return
		}
		//f(1)
	}
	testCallbacks(&cbTest{
		t:               t,
		ips:             ips,
		count:           0,
		timeout:         time.Second,
		interval:        time.Millisecond,
		expire:          5 * time.Second,
		countMultiplier: 0,
		cb:              cb,
	})
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
	testCallbacks(&cbTest{
		t:               t,
		ips:             ips,
		count:           4,
		timeout:         time.Second,
		interval:        time.Second,
		countMultiplier: 1,
		cb:              cb,
	})
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
	testCallbacks(&cbTest{
		t:               t,
		ips:             ips,
		count:           4,
		timeout:         time.Second,
		interval:        time.Second,
		countMultiplier: 1,
		setup:           setup,
		cb:              cb,
	})
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
	testCallbacks(&cbTest{
		t:               t,
		ips:             ips,
		count:           4,
		timeout:         time.Second,
		interval:        time.Second,
		countMultiplier: 2,
		setup:           setup,
		cb:              cb,
	})
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
	testCallbacks(&cbTest{
		t:               t,
		ips:             ips,
		count:           4,
		timeout:         time.Second,
		interval:        time.Second,
		countMultiplier: 2,
		setup:           setup,
		cb:              cb,
	})
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
	testCallbacks(&cbTest{
		t:               t,
		ips:             ips,
		count:           4,
		timeout:         time.Second,
		interval:        time.Second,
		countMultiplier: 2,
		setup:           setup,
		cb:              cb,
	})
}
