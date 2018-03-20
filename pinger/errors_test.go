package pinger

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/TrilliumIT/go-multiping/internal/listenmap"
)

func TestDupListener(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(DefaultConn().lm.Add(context.Background(), net.ParseIP("127.0.0.1"), 5, nil))
	assert.Equal(listenmap.ErrAlreadyExists, DefaultConn().lm.Add(context.Background(), net.ParseIP("127.0.0.1"), 5, nil))
}
