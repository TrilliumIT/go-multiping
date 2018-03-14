package pinger

import (
	//	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
	//"github.com/clinta/go-multiping/packet"
)

/*
func TestMain(m *testing.M) {
	go func() {
		time.Sleep(10 * time.Second)
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		panic("die")
	}()
	os.Exit(m.Run())
}
*/

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

func testNoCallbacks(t *testing.T, ips ...string) {
	igr := runtime.NumGoroutine()
	wg := sync.WaitGroup{}
	for _, ip := range ips {
		d := NewDst(ip, time.Second, time.Second, 2)
		wg.Add(1)
		go func() {
			defer wg.Done()
			checkErr(t, d.Run())
		}()
	}
	wg.Wait()
	checkGoRoutines(t, igr)
}

func testOnReply(t *testing.T, ips ...string) {
	igr := runtime.NumGoroutine()
	ti := 0
	for _, ip := range ips {
		d := NewDst(ip, time.Second, time.Second, 2)
		err := d.Run()
		if err == nil {
			t.Errorf("unexpected success")
		}
		checkGoRoutines(t, igr)
		/*
			i := 0
			d.SetOnReply(func(p *packet.Packet) { i++; ti++; fmt.Println("got packet") })
			checkErr(t, d.Run())
			if i != 2 {
				t.Error("all packets were not recieved")
			}
		*/
	}
	if ti != 2*len(ips) {
		t.Error("all packets were not recieved")
	}
	time.Sleep(time.Millisecond)
	checkGoRoutines(t, igr)
}

func TestNoCallbacksV4(t *testing.T) {
	testNoCallbacks(t, "127.0.0.1", "127.0.0.2", "127.0.0.3")
}

func testOnReplyv4(t *testing.T) {
	testOnReply(t, "127.0.0.1", "127.0.0.2", "127.0.0.3")
}

func testOnReplyv6(t *testing.T) {
	testOnReply(t, "::1", "::1", "::1")
}
