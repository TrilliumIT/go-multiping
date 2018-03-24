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

func TestSeqBlock(t *testing.T) {
	assert := assert.New(t)
	ip, err := net.ResolveIPAddr("ip", "192.0.2.1")
	assert.NoError(err)
	assert.NotNil(ip)
	h := func(*Ping, error) {}
	c, err := NewIPConn(ip, h, 0)
	assert.NoError(err)
	assert.NotNil(c)
	sc := make(chan struct{})
	wg := sync.WaitGroup{}
	for i := 0; i < 1<<16; i++ {
		// these shouldn't block
		wg.Add(1)
		go func() { c.SendPing(); sc <- struct{}{}; wg.Done() }()
		tm := time.NewTimer(500 * time.Millisecond)
		select {
		case <-tm.C:
			assert.Fail(fmt.Sprintf("sending blocked at %v", i))
		case <-sc:
		}
		tm.Stop()
	}
	// this should block
	go func() { c.SendPing(); sc <- struct{}{} }()
	tm := time.NewTimer(500 * time.Millisecond)
	select {
	case <-sc:
		assert.Fail("sending did not block")
	case <-tm.C:
	}
	tm.Stop()
	assert.NoError(c.Close())

	wgCh := make(chan struct{})
	go func() { wg.Wait(); close(wgCh) }()
	tm = time.NewTimer(500 * time.Millisecond)
	select {
	case <-tm.C:
		assert.Fail("close did not unblock pending sends")
	case <-wgCh:
	}
	tm.Stop()
}
