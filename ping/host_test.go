package ping

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHostOnceSuccess(t *testing.T) {
	assert := assert.New(t)
	host := "google.com"
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
