package pinger

import (
	//	"fmt"
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
		time.Sleep(10 * time.Second)
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

func TestNoCallbacks(t *testing.T) {
	testIPs := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	igr := runtime.NumGoroutine()
	wg := sync.WaitGroup{}
	for _, ip := range testIPs {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			d := NewDst(ip, time.Second, time.Second, 2)
			checkErr(t, d.Run())
		}(ip)
	}
	wg.Wait()
	checkGoRoutines(t, igr)
}

func testCallbacks(
	t *testing.T,
	ips []string,
	count int,
	setup func(d *Dst, f func()),
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
			if i != count {
				t.Errorf("only %v of %v packets counted", i, count)
			}
		}(ip)
	}
	wg.Wait()
	if ti != count*len(ips) {
		t.Errorf("only %v of %v total packets counted", ti, count*len(ips))
	}
	time.Sleep(time.Millisecond)
	checkGoRoutines(t, igr)
}

func TestOnReply(t *testing.T) {
	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}
	setup := func(d *Dst, f func()) {
		d.SetOnReply(func(*packet.Packet) { f() })
	}
	testCallbacks(t, ips, 4, setup)
}

func TestOnTimeout(t *testing.T) {
	var ips = []string{"192.0.2.0", "198.51.100.0", "203.0.113.0", "fe80::2", "fe80::3", "fe80::4"}
	setup := func(d *Dst, f func()) {
		d.SetOnTimeout(func(*packet.SentPacket) { f() })
	}
	testCallbacks(t, ips, 4, setup)
}

func TestOnSendError(t *testing.T) {
	var ips = []string{"0.0.0.1", "::2", "0.0.0.5", "::5"}
	setup := func(d *Dst, f func()) {
		d.SetOnSendError(func(*packet.SentPacket, error) { f() })
	}
	testCallbacks(t, ips, 4, setup)
}
