package pinger

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/TrilliumIT/go-multiping/internal/listenMap"
	"github.com/TrilliumIT/go-multiping/ping"
)

func TestDupListener(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(DefaultConn().lm.Add(context.Background(), net.ParseIP("127.0.0.1"), 5, nil))
	assert.Equal(DefaultConn().lm.Add(context.Background(), net.ParseIP("127.0.0.1"), 5, nil), listenMap.ErrAlreadyExists)
}

type counter struct {
	i int
	l sync.Mutex
}

func (c *counter) add() {
	c.l.Lock()
	c.i++
	c.l.Unlock()
}

func (c *counter) get() int {
	c.l.Lock()
	defer c.l.Unlock()
	return c.i
}

func testPingsWithCount(t *testing.T, c *PingConf) {
	t.Parallel()
	assert := assert.New(t)
	r := &counter{}
	h := func(ctx context.Context, p *ping.Ping, err error) {
		assert.NoError(err)
		assert.NotNil(p)
		r.add()
	}
	assert.NoError(Ping("127.0.0.1", h, c))
	assert.Equal(r.get(), c.Count)
}

func testPingsWithCancel(t *testing.T, c *PingConf, cancelAfter int) {
	t.Parallel()
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	r := &counter{}
	h := func(ctx context.Context, p *ping.Ping, err error) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		assert.NoError(err)
		assert.NotNil(p)
		r.add()
		if r.get() >= cancelAfter {
			cancel()
		}
	}
	assert.NoError(PingWithContext(ctx, "127.0.0.1", h, c))
	assert.Equal(r.get(), cancelAfter)
}

func Test1Ping(t *testing.T) {
	c := DefaultPingConf()
	c.Count = 1
	testPingsWithCount(t, c)
}

func Test1PingC(t *testing.T) {
	c := DefaultPingConf()
	c.Count = 0
	testPingsWithCancel(t, c, 1)
}

func Test10Pings(t *testing.T) {
	c := DefaultPingConf()
	c.Count = 0
	c.Interval = time.Millisecond
	testPingsWithCancel(t, c, 10)
}

func TestDNSError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	c := DefaultPingConf()
	_, ok := Ping("foo.test", nil, c).(*net.DNSError)
	assert.True(ok)
}

func TestDNSErrorRetry(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	c := DefaultPingConf()
	c.Count = 3
	c.RetryOnResolveError = true
	r := &counter{}
	h := func(ctx context.Context, p *ping.Ping, err error) {
		_, ok := err.(*net.DNSError)
		assert.True(ok)
		assert.NotNil(p)
		r.add()
	}
	assert.NoError(Ping("foo.test", h, c))
	assert.Equal(r.get(), c.Count)
}
