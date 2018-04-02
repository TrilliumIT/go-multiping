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

func TestIPOnceV4(t *testing.T) {
	assert := assert.New(t)

	dst, err := net.ResolveIPAddr("ip", "127.0.0.1")
	assert.NoError(err)
	assert.NotNil(dst)
	p, err := IPOnce(dst, time.Second)
	assert.NoError(err)
	assert.NotNil(p)
	assert.WithinDuration(time.Now(), p.Recieved, time.Second)
	assert.NotZero(p.RTT())
	assert.NotZero(p.TTL)
	assert.True(dst.IP.Equal(p.Dst.IP))
}

func TestIPOnceV6(t *testing.T) {
	assert := assert.New(t)

	dst, err := net.ResolveIPAddr("ip", "::1")
	assert.NoError(err)
	assert.NotNil(dst)
	p, err := IPOnce(dst, time.Second)
	assert.NoError(err)
	assert.NotNil(p)
	assert.WithinDuration(time.Now(), p.Recieved, time.Second)
	assert.WithinDuration(p.Sent, p.Recieved, time.Second)
	assert.NotZero(p.RTT())
	assert.NotZero(p.TTL)
	assert.True(dst.IP.Equal(p.Dst.IP))
}

func TestIPOnceUnreachable(t *testing.T) {
	assert := assert.New(t)
	dst, err := net.ResolveIPAddr("ip", "198.51.100.1")
	assert.NoError(err)
	p, err := IPOnce(dst, time.Second)
	assert.Equal(ErrTimedOut, err)
	assert.NotNil(p)
	assert.WithinDuration(time.Now(), p.Sent, 2*time.Second)
	assert.Zero(p.Recieved)
	assert.Zero(p.RTT())
	assert.Zero(p.TTL)
	assert.True(dst.IP.Equal(p.Dst.IP))
}

func testIPDrain(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	var received int64
	h := func(p *Ping, err error) {
		atomic.AddInt64(&received, 1)
		assert.NoError(err)
		assert.NotNil(p)
		assert.Equal(int64(1), atomic.LoadInt64(&received), "unexpected ping for %p", p)
	}
	dst, err := net.ResolveIPAddr("ip", "127.0.0.1")
	assert.NoError(err)
	ipc, err := NewIPConn(dst, h, time.Second)
	assert.NoError(err)
	p, err := ipc.getNextPing()
	ipc.sendPing(p, err)
	ipc.Drain()
	assert.Equal(int64(1), atomic.LoadInt64(&received), "missed ping for %p", p)
	assert.NoError(ipc.Close())
}

func TestIPDrain(t *testing.T) {
	for i := 0; i < 1000; i++ {
		t.Run(fmt.Sprintf("%v", i), testIPDrain)
	}
}

func Test10kDeadLock(t *testing.T) {
	assert := assert.New(t)
	var received int64
	h := func(p *Ping, err error) {
		atomic.AddInt64(&received, 1)
		assert.NoError(err)
		assert.NotNil(p)
	}
	dst, err := net.ResolveIPAddr("ip", "127.0.0.1")
	assert.NoError(err)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Second, func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		assert.FailNow("interval took too long")
		cancel()
	})
	assert.NoError(IPInterval(ctx, dst, h, 10000, 0, time.Second))
	cancel()
}
