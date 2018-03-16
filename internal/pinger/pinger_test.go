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
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// nolint:dupl
func TestNewV4(t *testing.T) {
	p := New(4)
	if p.callbacks == nil {
		t.Error("callbacks should not be nil")
	}
	if p.Network() != "ip4:icmp" {
		t.Error("wrong network")
	}
	if p.src != "0.0.0.0" {
		t.Error("wrong source")
	}
	if p.SendType() != ipv4.ICMPTypeEcho {
		t.Error("wrong send type")
	}
	if p.expectedLen != 16+v4AddLen {
		fmt.Println(p.expectedLen)
		t.Error("wrong expected length")
	}
}

// nolint:dupl
func TestNewV6(t *testing.T) {
	p := New(6)
	if p.callbacks == nil {
		t.Error("callbacks should not be nil")
	}
	if p.Network() != "ip6:ipv6-icmp" {
		t.Error("wrong network")
	}
	if p.src != "::" {
		t.Error("wrong source")
	}
	if p.SendType() != ipv6.ICMPTypeEchoRequest {
		t.Error("wrong send type")
	}
	if p.expectedLen != 16+v6AddLen {
		fmt.Println(p.expectedLen)
		t.Error("wrong expected length")
	}
}

func TestNewInvalidV(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("invalid pinger should panic")
		}
	}()
	_ = New(5)
}

func TestAddCallBackNilIP(t *testing.T) {
	cb := func(*ping.Ping) {}
	p := New(4)
	err := p.AddCallBack(nil, 123, cb)
	if err == nil {
		t.Error("expected error with nil ip")
	}
	_, ok := p.GetCallback(net.ParseIP("127.0.0.1"), 123)
	if ok {
		t.Error("expected not ok getting callback")
	}
}

func TestAddCallBackNilCB(t *testing.T) {
	var cb func(*ping.Ping)
	p := New(4)
	err := p.AddCallBack(net.ParseIP("127.0.0.1"), 123, cb)
	if err == nil {
		t.Error("expected error with nil cb")
	}
	_, ok := p.GetCallback(net.ParseIP("127.0.0.1"), 123)
	if ok {
		t.Error("expected not ok getting callback")
	}
}

func addCb(t *testing.T, p *Pinger, ip string, id int, cb func(*ping.Ping), irt int) {
	err := p.AddCallBack(net.ParseIP(ip), id, cb)
	if err != nil {
		t.Error("unexpected error with valid callback")
	}
	rcb, ok := p.GetCallback(net.ParseIP(ip), id)
	if !ok {
		t.Error("expected ok getting callback")
	}
	if rcb == nil {
		t.Error("expected callback function")
	}
	if runtime.NumGoroutine() < irt+2 {
		fmt.Println(runtime.NumGoroutine() - irt)
		t.Error("listener is not running")
	}
}

func delCB(t *testing.T, p *Pinger, ip string, id int) {
	err := p.DelCallBack(net.ParseIP(ip), id)
	if err != nil {
		t.Error("unexpected error deleting valid callback")
	}
}

func testCallBacks(t *testing.T, proto int, ip string, id1, id2 int) {
	initGoRoutines := runtime.NumGoroutine()
	l := sync.Mutex{}
	l.Lock()
	r1, r2 := 0, 0
	l.Unlock()
	cb1 := func(p *ping.Ping) {
		l.Lock()
		defer l.Unlock()
		r1 = p.Seq
	}
	cb2 := func(p *ping.Ping) {
		l.Lock()
		defer l.Unlock()
		r2 = p.Seq
	}
	p := New(proto)
	addCb(t, p, ip, id1, cb1, initGoRoutines)
	singleListenerGoRoutines := runtime.NumGoroutine()
	sendTo(t, p, ip, id1, 1)
	sendTo(t, p, ip, id2, 1)
	time.Sleep(500 * time.Millisecond)
	l.Lock()
	if r1 != 1 || r2 != 0 {
		t.Errorf("wrong recieved packet count: %v and %v", r1, r2)
	}
	l.Unlock()
	_, ok := p.AddCallBack(net.ParseIP(ip), id1, cb1).(*ErrorAlreadyExists)
	if !ok {
		t.Error("expected error on dupliate callback")
	}
	addCb(t, p, ip, id2, cb2, initGoRoutines)
	if runtime.NumGoroutine() != singleListenerGoRoutines {
		fmt.Println(runtime.NumGoroutine() - singleListenerGoRoutines)
		t.Error("listeners changing")
	}
	sendTo(t, p, ip, id1, 2)
	sendTo(t, p, ip, id2, 2)
	time.Sleep(500 * time.Millisecond)
	l.Lock()
	if r1 != 2 || r2 != 2 {
		t.Errorf("wrong recieved packet count: %v and %v", r1, r2)
	}
	l.Unlock()
	delCB(t, p, ip, id1)
	if runtime.NumGoroutine() != singleListenerGoRoutines {
		fmt.Println(runtime.NumGoroutine() - singleListenerGoRoutines)
		t.Error("listeners changing")
	}
	sendTo(t, p, ip, id1, 3)
	sendTo(t, p, ip, id2, 3)
	time.Sleep(500 * time.Millisecond)
	l.Lock()
	if r1 != 2 || r2 != 3 {
		t.Errorf("wrong recieved packet count: %v and %v", r1, r2)
	}
	l.Unlock()
	delCB(t, p, ip, id2)
	time.Sleep(time.Millisecond)
	if runtime.NumGoroutine() > initGoRoutines {
		fmt.Println(runtime.NumGoroutine() - initGoRoutines)
		_ = pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		t.Error("goroutines leaking")
	}
	l.Lock()
	if r1 != 2 || r2 != 3 {
		t.Errorf("wrong recieved packet count: %v and %v", r1, r2)
	}
	l.Unlock()
}

func TestCallBacksv4(t *testing.T) {
	testCallBacks(t, 4, "127.0.0.1", 1, 2)
}

func TestCallBacksv6(t *testing.T) {
	testCallBacks(t, 6, "::1", 1, 2)
}

func sendTo(t *testing.T, pp *Pinger, ip string, id, seq int) {
	dst, err := net.ResolveIPAddr("ip", ip)
	p := &ping.Ping{Dst: dst.IP, ID: id, Seq: seq}
	if err != nil {
		t.Error("unexpected error from resolve")
	}
	err = pp.Send(dst, p, time.Second)
	if err != nil {
		t.Errorf("unexpected error from send: %v", err)
	}
}

func TestListenFailure(t *testing.T) {
	p := New(4)
	p.src = "::1"
	f, err := p.listen()
	if err == nil {
		t.Errorf("expected error for listen with invalid src/network combination")
	}
	err, wait := f()
	if err != nil {
		t.Errorf("expected nil error from f")
	}
	wait()
}
