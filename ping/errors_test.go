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

func assertBlocks(t *testing.T, f func(), to time.Duration, s string, i ...interface{}) {
	tm := time.NewTimer(to)
	r := make(chan struct{})
	go func() { f(); close(r) }()
	select {
	case <-tm.C:
	case <-r:
		assert.Fail(t, s, i...)
	}
	tm.Stop()
}

func assertDoesNotBlock(t *testing.T, f func(), to time.Duration, s string, i ...interface{}) {
	tm := time.NewTimer(to)
	r := make(chan struct{})
	go func() { f(); close(r) }()
	select {
	case <-tm.C:
		assert.Fail(t, s, i...)
	case <-r:
	}
	tm.Stop()
}

func TestSeqBlock(t *testing.T) {
	assert := assert.New(t)
	ip, err := net.ResolveIPAddr("ip", "192.0.2.1")
	assert.NoError(err)
	assert.NotNil(ip)
	h := func(p *Ping, err error) {
		fmt.Printf("%#v\n", p)
		fmt.Printf("%#v\n", err)
		assert.FailNow("bah")

		assert.Equal(ErrNotRunning, err)
		assert.Equal(1<<16, p.Count)
	}
	c, err := NewIPConn(ip, h, 0)
	assert.NoError(err)
	assert.NotNil(c)
	wg := sync.WaitGroup{}
	wg.Add(1 << 16)
	for i := 0; i < 1<<16; i++ {
		f := func() { assert.Equal(i, c.SendPing()); wg.Done() }
		// these shouldn't block
		assertDoesNotBlock(t, f, 10*time.Second, fmt.Sprintf("ping %v blocked", i))
	}
	time.Sleep(time.Second)
	// this should block
	wg2 := sync.WaitGroup{}
	wg2.Add(1)
	f := func() { fmt.Println("lws"); assert.Equal(1<<16, c.SendPing()); fmt.Println("lwe"); wg2.Done() }
	assertBlocks(t, f, time.Second, "last ping did not block")
	fmt.Println("1")
	assert.NoError(c.Close())

	assertDoesNotBlock(t, wg.Wait, 500*time.Millisecond, "first pings did not return")
	fmt.Println("2")
	assertDoesNotBlock(t, wg2.Wait, 2000*time.Millisecond, "last ping did not return")
}
