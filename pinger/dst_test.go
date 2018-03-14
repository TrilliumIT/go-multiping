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

var testIPs = []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "::1", "::1", "::1"}

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

func testNoCallbacks(t *testing.T) {
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

func TestOnReply(t *testing.T) {
	igr := runtime.NumGoroutine()
	ti := 0
	addTi := sync.Mutex{}
	wg := sync.WaitGroup{}
	for _, ip := range testIPs {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			d := NewDst(ip, time.Second, time.Second, 2)
			i := 0
			d.SetOnReply(func(p *packet.Packet) {
				addTi.Lock()
				defer addTi.Unlock()
				i++
				ti++
			})
			checkErr(t, d.Run())
			if i != 2 {
				t.Errorf("only %v of %v packets recieved", i, 2)
			}
		}(ip)
	}
	wg.Wait()
	if ti != 2*len(testIPs) {
		t.Error("all packets were not recieved")
	}
	time.Sleep(time.Millisecond)
	checkGoRoutines(t, igr)
}
