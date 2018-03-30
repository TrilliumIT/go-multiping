package seqmap

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
)

func TestDrain(t *testing.T) {
	assert := assert.New(t)
	var received int64
	sm := New(func(p *ping.Ping, err error) {
		time.Sleep(10 * time.Millisecond)
		assert.NoError(err)
		assert.NotNil(p)
		assert.Equal(int64(1), atomic.AddInt64(&received, 1))
	})
	p := &ping.Ping{Seq: 0}
	assert.Equal(1, sm.Add(p))
	go func() {
		rp, l, err := sm.Pop(0)
		assert.Equal(0, l)
		assert.NoError(err)
		assert.NotNil(rp)
		sm.Handle(rp, err)
	}()
	sm.Drain() // this should block until handle is done
	assert.Equal(int64(1), atomic.LoadInt64(&received))
}
