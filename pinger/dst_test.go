package pinger

import (
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/clinta/go-multiping/packet"
)

func checkGoRoutines(t *testing.T, igr int) {
	gr := runtime.NumGoroutine()
	if gr > igr {
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		t.Errorf("leaking goroutines: %v", gr-igr)
	}
}

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func testOnReply(t *testing.T, ips ...string) {
	igr := runtime.NumGoroutine()
	ti := 0
	for _, ip := range ips {
		d := NewDst(ip, 100*time.Millisecond, time.Second, 2)
		err := d.Run()
		if err == nil {
			t.Errorf("unexpected success")
		}
		i := 0
		d.SetOnReply(func(p *packet.Packet) { i++; ti++ })
		checkErr(t, d.Run())
		if i != 2 {
			t.Error("all packets were not recieved")
		}
	}
	if ti != 2*len(ips) {
		t.Error("all packets were not recieved")
	}
	time.Sleep(time.Millisecond)
	checkGoRoutines(t, igr)
}

func TestOnReplyv4(t *testing.T) {
	testOnReply(t, "127.0.0.1", "127.0.0.2", "127.0.0.3")
}

func TestOnReplyv6(t *testing.T) {
	testOnReply(t, "::1", "::1", "::1")
}
