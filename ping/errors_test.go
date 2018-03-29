package ping

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNoIDs(t *testing.T) {
	assert := assert.New(t)
	conns := make([]*IPConn, 1<<16, 1<<16)
	assert.Equal(len(conns), 1<<16)
	ip, err := net.ResolveIPAddr("ip", "127.0.0.1")
	assert.NoError(err)
	assert.NotNil(ip)
	h := func(*Ping, error) {}
	for i := range conns {
		conns[i], err = NewIPConn(ip, h, 0)
		assert.NoError(err)
		assert.NotNil(conns[i])
	}
	var c *IPConn
	c, err = NewIPConn(ip, h, 0)
	assert.Equal(ErrNoIDs, err)
	assert.Nil(c)
	for _, c := range conns {
		assert.NoError(c.Close())
	}
}

func assertBlocks(t *testing.T, f func(), minD, maxD time.Duration, s string, i ...interface{}) {
	tm := time.AfterFunc(maxD*2, func() {
		panic("blocked twice timeout limit")
	})
	st := time.Now()
	f()
	tm.Stop()
	assert.True(t, time.Now().Sub(st) > minD, "Did not block long enough")
}

func assertDoesNotBlock(t *testing.T, f func(), maxD time.Duration, s string, i ...interface{}) {
	tm := time.AfterFunc(maxD*2, func() {
		panic("blocked twice timeout limit")
	})
	st := time.Now()
	f()
	tm.Stop()
	assert.WithinDuration(t, st, time.Now(), maxD)
}

func TestSeqBlock(t *testing.T) {
	assert := assert.New(t)
	ip, err := net.ResolveIPAddr("ip", "192.0.2.1")
	assert.NoError(err)
	assert.NotNil(ip)
	h := func(p *Ping, err error) {
		assert.Fail(
			fmt.Sprintf(`Handle should have not been called.\n
			Called with packet: %#v\n
			And error: %#v\n
			And error(): %v`, p, err, err.Error()))
	}
	c, err := NewIPConn(ip, h, 0)
	assert.NoError(err)
	assert.NotNil(c)
	wg := sync.WaitGroup{}
	wg.Add(1 << 16)
	for i := 0; i < 1<<16; i++ {
		f := func() { c.SendPing(); wg.Done() }
		// these shouldn't block
		assertDoesNotBlock(t, f, 10*time.Second, fmt.Sprintf("ping %v blocked", i))
	}
	time.Sleep(time.Second)
	// this should block
	wg2 := sync.WaitGroup{}
	wg2.Add(1)
	f := func() {
		c.SendPing()
		wg2.Done()
	}
	go assertBlocks(t, f, time.Second, 10*time.Second, "last ping did not block")
	time.Sleep(2 * time.Second)
	assert.NoError(c.Close())

	assertDoesNotBlock(t, wg.Wait, 500*time.Millisecond, "first pings did not return")
	assertDoesNotBlock(t, wg2.Wait, 2000*time.Millisecond, "last ping did not return")
}
