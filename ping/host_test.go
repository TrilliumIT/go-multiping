package ping

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHostOnceSuccess(t *testing.T) {
	assert := assert.New(t)
	host := "example.com"
	timeout := time.Second

	p, err := HostOnce(host, timeout)
	assert.NoError(err)
	assert.NotNil(p)
	assert.WithinDuration(p.Recieved, p.Sent, timeout)
	assert.WithinDuration(p.Recieved, time.Now(), time.Millisecond)
	assert.Equal(p.Recieved.Sub(p.Sent), p.RTT())
	assert.Equal(timeout, p.TimeOut)
	assert.Equal(p.Sent.Add(timeout), p.TimeOutTime())
	assert.True(p.RTT() < time.Second)
	assert.Equal(host, p.Host)
	if assert.NotNil(p.Dst) {
		assert.NotNil(p.Dst.IP)
	}
	if assert.NotNil(p.Dst) {
		assert.NotNil(p.Dst.IP)
	}
	assert.Equal(0, p.Seq)
	assert.True(0 <= p.ID || p.ID < 1<<16)
	assert.True(p.TTL > 0)
	assert.True(p.Len > 0)
}

func TestHostOnceFail(t *testing.T) {
	assert := assert.New(t)
	host := "foo.test"
	timeout := 10 * time.Millisecond

	p, err := HostOnce(host, timeout)
	assert.Error(err)
	_, ok := err.(*net.DNSError)
	assert.True(ok)
	assert.NotNil(p)
	assert.Equal(host, p.Host)
	assert.Nil(p.Dst)
	assert.WithinDuration(time.Now(), p.Sent, timeout*5)
}

func testSuccess(host string, reResolveEvery int, count int, interval, timeout time.Duration) func(*testing.T) {
	return func(t *testing.T) {
		assert := assert.New(t)
		var received int64
		hf := func(p *Ping, err error) {
			assert.NoError(err)
			atomic.AddInt64(&received, 1)
			assert.NotNil(p)
			assert.NotZero(p.RTT())
			assert.NotZero(p.Recieved)
			assert.True(p.Dst.IP.Equal(net.ParseIP(host)))
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		expTm := time.NewTimer(interval*time.Duration(count) + 3*timeout)
		go func() {
			select {
			case <-done:
				return
			case <-expTm.C:
				cancel()
				assert.FailNow("interval test did not complete in time")
			}
		}()
		assert.NoError(HostInterval(ctx, host, reResolveEvery, hf, count, interval, timeout))
		close(done)
		time.Sleep(timeout)
		assert.Equal(int64(count), atomic.LoadInt64(&received))
	}
}

func TestHostSuccess(t *testing.T) {
	intervals := []time.Duration{
		time.Nanosecond,
		time.Microsecond,
		100 * time.Microsecond,
		time.Millisecond,
		time.Second,
	}
	counts := []int{
		1,
		5,
		10,
		20,
	}
	hosts := []string{
		"127.0.0.1",
		"127.0.0.2",
		"localhost",
		"::1",
	}
	for _, h := range hosts {
		h := h
		t.Run(fmt.Sprintf("host-%v", h), func(t *testing.T) {
			for _, c := range counts {
				c := c
				t.Run(fmt.Sprintf("count-%v", c), func(t *testing.T) {
					for _, i := range intervals {
						i := i
						t.Run(fmt.Sprintf("int-%v", i), testSuccess(h, 0, c, i, time.Second))
					}
				})
			}
		})
	}
}
