package packet

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestToSentPacketNil(t *testing.T) {
	var p *Packet
	s := p.ToSentPacket()
	if s != nil {
		t.Error("Expected nil sent packet")
	}
}

func TestToSentPacket(t *testing.T) {
	p := &Packet{}
	p.Src = net.ParseIP("127.0.0.2")
	p.ID = 1234
	p.Seq = 5
	p.Sent = time.Now()

	s := p.ToSentPacket()
	if !s.Dst.Equal(p.Src) {
		t.Error("Expected equal dst")
	}
	if s.ID != p.ID {
		t.Error("Expected equal id")
	}
	if s.Seq != p.Seq {
		t.Error("Expected equal seq")
	}
	if !s.Sent.Equal(p.Sent) {
		t.Error("Expected equal sent")
	}
}

func TestBytesToTimeTooShort(t *testing.T) {
	b := make([]byte, TimeSliceLength-1)
	_, err := BytesToTime(b)
	if err == nil {
		t.Errorf("Expected error from too short bytes")
	}
}

func testValidTime(t *testing.T, tm time.Time) {
	b := TimeToBytes(tm)
	nt, err := BytesToTime(b)
	if err != nil {
		t.Error("Expected no error from BytesToTime")
	}
	if !nt.Equal(tm) {
		t.Errorf("Expected %v equal %v", tm, nt)
	}
}

func TestTimeToBytesNow(t *testing.T) {
	testValidTime(t, time.Now())
}

func TestTimeToBytesZero(t *testing.T) {
	tm := time.Unix(0, 0)
	if !bytes.Equal(TimeToBytes(tm), make([]byte, TimeSliceLength)) {
		t.Error("Expected zero time to be empty slice")
	}
	testValidTime(t, tm)
}

func TestTimeToBytesOne(t *testing.T) {
	tm := time.Unix(0, 1)
	testValidTime(t, tm)
}

func TestTimeToBytesMax(t *testing.T) {
	tm := time.Unix(0, 1<<63-1)
	testValidTime(t, tm)
}
