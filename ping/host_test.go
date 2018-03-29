package ping

import (
	"context"
	"net"
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

func testHostIntervalSuccess(host string, reResolveEvery int, count int, interval, timeout time.Duration) func(*testing.T) {
	return func(t *testing.T) {
		assert := assert.New(t)
		hf := func(p *Ping, err error) {
			assert.NoError(err)
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
	}
}

func TestHostInterval(t *testing.T) {
	t.Run("TestHostInterval", testHostIntervalSuccess("127.0.0.1", 0, 100, 10*time.Microsecond, time.Second))
}
