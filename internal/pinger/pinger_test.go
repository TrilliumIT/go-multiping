package pinger

import (
	"fmt"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"testing"
)

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
	if p.expectedLen != 16 {
		fmt.Println(p.expectedLen)
		t.Error("wrong expected length")
	}
}

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
	if p.expectedLen != 16 {
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
