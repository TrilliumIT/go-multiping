package ping

import (
	"net"
	"testing"
	//"sync/atomic"

	"github.com/stretchr/testify/assert"
)

func TestIPOnce(t *testing.T) {
	assert := assert.New(t)

	dst, err := net.ResolveIPAddr("ip", "127.0.0.1")
	assert.NoError(err)
	assert.NotNil(dst)
}
