package ping

import (
	"bytes"
	"testing"
	"time"
)

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
