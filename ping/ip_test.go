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

func TestIPOnce(t *testing.T) {
	assert := assert.New(t)

	dst, err := net.ResolveIPAddr("ip", "127.0.0.1")
	assert.NoError(err)
	assert.NotNil(dst)
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
