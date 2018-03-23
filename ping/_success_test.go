package ping

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/TrilliumIT/go-multiping/ping"
)

func TestMain(m *testing.M) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		go func() {
			err := pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
			time.Sleep(1 * time.Second)
			panic(err)
		}()
	}()
	os.Exit(m.Run())
}

const addTimeout = 5 * time.Second

func testPingSuccess(t *testing.T, h string, cf PingConf, cancelAfter int) {
	assert := assert.New(t)
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
	r := 0
	rL := sync.Mutex{}
	hf := func(ctx context.Context, p *ping.Ping, err error) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		rL.Lock()
		defer rL.Unlock()
		select {
		case <-ctx.Done():
			return
		default:
		}
		assert.NoError(err)
		assert.NotNil(p)
		r++
		if cancelAfter != 0 && r >= cancelAfter {
			cancel()
		}
	}
	assert.NoError(PingWithContext(ctx, h, hf, &cf))
	if cf.Count != 0 {
		assert.Equal(cf.Count, r)
	}
	if cancelAfter != 0 {
		assert.Equal(cancelAfter, r)
	}
}

func testTypes(t *testing.T, cf PingConf, h string) {
	t.Parallel()
	cancelAfter := 0
	t.Run("TestCount", func(t *testing.T) { testPingSuccess(t, h, cf, cancelAfter) })
	cancelAfter = cf.Count
	cf.Count = 0
	t.Run("TestCancel", func(t *testing.T) { testPingSuccess(t, h, cf, cancelAfter) })
}

func testHosts(t *testing.T, cf PingConf) {
	t.Parallel()
	hosts := []string{
		"127.0.0.1",
		"127.0.0.1",
		//"::1",
		//"::1",
		"127.0.0.2",
		"127.0.0.3",
	}

	for _, h := range hosts {
		n := fmt.Sprintf("Host:%v", h)
		t.Run(n, func(t *testing.T) { testTypes(t, cf, h) })
	}
}

func testIntervals(t *testing.T, cf PingConf) {
	t.Parallel()
	intervals := []time.Duration{
		0,
		time.Nanosecond,
		time.Microsecond,
		time.Millisecond,
		20 * time.Millisecond,
	}
	for _, i := range intervals {
		n := fmt.Sprintf("Interval:%v", i)
		cf.Interval = i
		t.Run(n, func(t *testing.T) { testHosts(t, cf) })
	}
}

func testCounts(t *testing.T, cf PingConf) {
	t.Parallel()
	counts := []int{
		1,
		2,
		10,
	}
	for _, c := range counts {
		n := fmt.Sprintf("Count:%v", c)
		cf.Count = c
		t.Run(n, func(t *testing.T) { testIntervals(t, cf) })
	}
}

func testWorkers(t *testing.T) {
	t.Parallel()
	workers := []int{
		-2,
		-1,
		0,
		1,
		2,
	}
	for _, w := range workers {
		n := fmt.Sprintf("Workers:%v", w)
		cf := *DefaultPingConf()
		cf.Workers = w
		t.Run(n, func(t *testing.T) { testCounts(t, cf) })
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
	for _, c := range conWorkers {
		n := fmt.Sprintf("ConnWorkers:%v", c)
		DefaultConn().SetWorkers(c)
		t.Run(n, testWorkers)
	}
}

func TestSuccesses(t *testing.T) {
	testConnWorkers(t)
}
