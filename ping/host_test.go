package ping

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"
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

func testInterval(host string, reResolveEvery int, count int, interval, timeout time.Duration) func(*testing.T) {
	return func(t *testing.T) {
		assert := assert.New(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancelOn := int64(count * 2)
		if count == 0 {
			cancelOn = 10
		}
		var received int64
		hf := func(p *Ping, err error) {
			select {
			case <-ctx.Done():
				debug.PrintStack()
				panic("late comer")
				return
			default:
			}
			r := atomic.AddInt64(&received, 1)
			if r >= cancelOn {
				cancel()
			}
			assert.NoError(err)
			assert.NotNil(p)
			assert.NotZero(p.RTT())
			assert.NotZero(p.Recieved)
		}
		done := make(chan struct{})
		expTm := time.NewTimer(interval*time.Duration(cancelOn) + 3*timeout)
		go func() {
			select {
			case <-done:
				return
			case <-expTm.C:
				cancel()
				assert.FailNow("interval test did not complete in time")
			}
		}()
		switch interval {
		case -1:
			assert.NoError(HostFlood(ctx, host, reResolveEvery, hf, count, timeout))
		default:
			assert.NoError(HostInterval(ctx, host, reResolveEvery, hf, count, interval, timeout))
		}
		close(done)
		cancel()
		switch count {
		case 0:
			assert.True(atomic.LoadInt64(&received) >= cancelOn)
		default:
			assert.Equal(int64(count), atomic.LoadInt64(&received))
		}
	}
}

func TestHostInterval(t *testing.T) {
	DefaultSocket().SetWorkers(4)
	intervals := []time.Duration{
		-1,
		0,
		time.Nanosecond,
		time.Microsecond,
		100 * time.Microsecond,
		time.Millisecond,
		time.Second,
	}
	counts := []int{
		//0,
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
			t.Parallel()
			for _, c := range counts {
				c := c
				t.Run(fmt.Sprintf("count-%v", c), func(t *testing.T) {
					for _, i := range intervals {
						i := i
						t.Run(fmt.Sprintf("int-%v", i), testInterval(h, 0, c, i, time.Second))
					}
				})
			}
		})
	}
}
