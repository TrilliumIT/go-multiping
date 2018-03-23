package ping

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/TrilliumIT/go-multiping/ping/internal/listenmap"
)

func TestDupListener(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	assert.NoError(DefaultConn().lm.Add(ctx, net.ParseIP("127.0.0.1"), 5, nil))
	assert.Equal(listenmap.ErrAlreadyExists, DefaultConn().lm.Add(context.Background(), net.ParseIP("127.0.0.1"), 5, nil))
	cancel()
}

/*

test filling ids
for i := 0; i < 1<<16
addconn(localhost)
add one more, check or error

test filling seq
for i to 1<<16
send to timeout host with to = 0
check that sending one more blocks
*/
