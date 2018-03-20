package pinger

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/TrilliumIT/go-multiping/internal/listenmap"
	"github.com/TrilliumIT/go-multiping/ping"
)

func TestDupListener(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(DefaultConn().lm.Add(context.Background(), net.ParseIP("127.0.0.1"), 5, nil))
	assert.Equal(listenmap.ErrAlreadyExists, DefaultConn().lm.Add(context.Background(), net.ParseIP("127.0.0.1"), 5, nil))
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

const addTimeout = 5 * time.Second

func testPingsWithCount(t *testing.T, cf *PingConf) {
	assert := assert.New(t)
	assert.NotEqual(0, cf.Count, "Count test should not have 0 count")
	to := time.Duration(cf.Count)*cf.Interval + addTimeout
	if !assert.True(to >= addTimeout && to < 3*addTimeout) {
		assert.FailNow("bad timeout")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(to, func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		assert.Fail("test took too long")
		cancel()
	})
	r := &counter{}
	h := func(ctx context.Context, p *ping.Ping, err error) {
		assert.NoError(err)
		assert.NotNil(p)
		r.add()
	}
	assert.NoError(PingWithContext(ctx, "127.0.0.1", h, cf))
	assert.Equal(cf.Count, r.get())
	cancel()
}

func testPingsWithCancel(t *testing.T, cf *PingConf) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelAfter := cf.Count
	assert.NotEqual(0, cancelAfter, "cancelAfter should be 0")
	to := time.Duration(cf.Count)*cf.Interval + addTimeout
	if !assert.True(to >= addTimeout && to < 3*addTimeout) {
		assert.FailNow("bad timeout")
	}
	cf.Count = 0
	time.AfterFunc(to, func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		assert.Fail("test took too long")
		cancel()
	})
	r := &counter{}
	h := func(ctx context.Context, p *ping.Ping, err error) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		r.l.Lock()
		defer r.l.Unlock()
		select {
		case <-ctx.Done():
			return
		default:
		}
		assert.NoError(err)
		assert.NotNil(p)
		r.i++
		if r.i >= cancelAfter {
			cancel()
		}
	}
	assert.NoError(PingWithContext(ctx, "127.0.0.1", h, cf))
	assert.Equal(cancelAfter, r.get())
}

func runTests(t *testing.T, cf *PingConf) {
	tests := map[string]func(*testing.T, *PingConf){
		"PingsWithCount":  testPingsWithCount,
		"PingsWithCancel": testPingsWithCancel,
	}
	for tn, tt := range tests {
		tcf := DefaultPingConf()
		tcf.Workers = cf.Workers
		tcf.Count = cf.Count
		tcf.Interval = cf.Interval
		t.Run(tn, func(t *testing.T) { tt(t, tcf) })
	}
}

func testIntervals(t *testing.T, cf *PingConf) {
	intervals := []time.Duration{
		0,
		time.Nanosecond,
		time.Microsecond,
		time.Millisecond,
		20 * time.Millisecond,
	}
	for _, i := range intervals {
		n := fmt.Sprintf("%vInterval", i)
		tcf := DefaultPingConf()
		tcf.Workers = cf.Workers
		tcf.Count = cf.Count
		tcf.Interval = i
		t.Run(n, func(t *testing.T) { runTests(t, tcf) })
	}
}

func testCounts(t *testing.T, cf *PingConf) {
	counts := []int{
		1,
		2,
		10,
	}
	for _, c := range counts {
		n := fmt.Sprintf("%vCount", c)
		tcf := DefaultPingConf()
		tcf.Workers = cf.Workers
		tcf.Count = c
		t.Run(n, func(t *testing.T) { testIntervals(t, tcf) })
	}
}

func testWorkers(t *testing.T) {
	workers := []int{
		-2,
		-1,
		0,
		1,
		2,
	}
	for _, w := range workers {
		n := fmt.Sprintf("%vWorkers", w)
		tcf := DefaultPingConf()
		tcf.Workers = w
		t.Run(n, func(t *testing.T) { testCounts(t, tcf) })
	}
}

func testConnWorkers(t *testing.T) {
	conWorkers := []int{
		-2,
		-1,
		0,
		1,
		2,
	}
	for _, cw := range conWorkers {
		n := fmt.Sprintf("Test%vConnWorkers", cw)
		DefaultConn().SetWorkers(cw)
		t.Run(n, testWorkers)
	}
}

func TestOptions(t *testing.T) {
	testConnWorkers(t)
}
