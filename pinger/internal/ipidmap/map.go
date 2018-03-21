package socket

import (
	"errors"
	"net"
	"sync"
	"time"

	pending "github.com/TrilliumIT/go-multiping/pinger/internal/pending2"
	"github.com/TrilliumIT/go-multiping/pinger/internal/seqmap"
)

type IpIdMap interface {
	// returns the length of the map after modification
	Add(net.IP, int) (*seqmap.SeqMap, int, error)
	// returns the length of the map after modification
	Pop(net.IP, int) (*seqmap.SeqMap, int, error)
	Get(net.IP, int) (*seqmap.SeqMap, bool)
}

var ErrAlreadyExists = errors.New("already exists")
var ErrDoesNotExist = errors.New("does not exist")

type timeoutMap struct {
	l sync.RWMutex
	m map[*pending.Ping]time.Time
}
