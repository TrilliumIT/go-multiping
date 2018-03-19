package pinger

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/TrilliumIT/go-multiping/ping"
	"github.com/TrilliumIT/go-multiping/pinger/internal/pending"
	"github.com/TrilliumIT/go-multiping/pinger/internal/ticker"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// PingConf holds optional coniguration parameters
type PingConf struct {
	// Count is how many pings will be attempted
	// 0 for unlimited pings
	Count int

	// Interval is the interval between pings
	// 0 for flood ping, sending a new packet as soon as the previous one is replied or times out
	Interval time.Duration

	// Timeout is how long to wait before considering a ping timed out
	Timeout time.Duration

	// RandDelay causes the first ping to be delayed by a random amount up to interval
	RandDelay bool

	// RetryOnResolveError will cause callback to be called on resolution errors
	// Resolution errors can be identied by error type net.DNSError
	// Otherwise Ping() will stop and retun an error on resolution errors
	RetryOnResolveError bool

	// RetryOnSendError will cause callback to be called on send errors.
	// Otherwise Ping() will stop and return an error on send errors.
	RetryOnSendError bool

	// ReResolveEvery caused pinger to re-resolve dns for a host every n pings.
	// 0 to disable. 1 to re-resolve every ping
	ReResolveEvery int

	// Workers is a number of workers to run handlers.
	// <-1 disables workers entirely and synchronously processes the incoming packet before listening for a new one.
	// This will prevent new pings from being able to be sent until the last one is handled, either returned or timed out.
	// -1 enables automatic worker allocation. Any time packet processing would block, a new worker is started. Workers remain active until all active pings stop. This is the default.
	// 0 disables workers, causing each incoming packet to start a new goroutine to handle it.
	// 0 and -1 can cause a memory leak while packets time out if timeout is less than interval. This can also cause a memory leak if it takes longer to run handler than interval.
	// A postive number pre-allocates a set number of workers
	Workers int

	// Buffer is the buffer length for handling packets. This can help handle bursts of pings without increasing workers.
	Buffer int
}

// DefaultPingConf returns a default ping configuration with an interval and timeout of 1 second
func DefaultPingConf() *PingConf {
	return &PingConf{
		Interval: time.Second,
		Timeout:  time.Second,
		Workers:  -1,
	}
}

func (p *PingConf) validate() *PingConf {
	if p == nil {
		p = DefaultPingConf()
	}
	return p
}

// ErrTimedOut is returned when a ping is timed out
var ErrTimedOut = pending.ErrTimedOut

// ErrSeqWrapped is returned if a packet is still being waited on by the time the ping sequence number wrapped and sent a new ping with this same sequence number
// This is only likely to happen if using a very short interval with a very long, or non-existent timeout
var ErrSeqWrapped = errors.New("response not recieved before sequence wrapped")

// HandleFunc is a function to handle pings
type HandleFunc func(context.Context, *ping.Ping, error)

// Ping starts a ping with a context that will be automatically canceled.
// See https://godoc.org/github.com/TrilliumIT/go-multiping/pinger#Conn.PingWithContext
func Ping(host string, hf HandleFunc, conf *PingConf) error {
	return DefaultConn().Ping(host, hf, conf)
}

// Ping starts a ping with a context that will be automatically canceled.
// See https://godoc.org/github.com/TrilliumIT/go-multiping/pinger#Conn.PingWithContext
func (c *Conn) Ping(host string, hf HandleFunc, conf *PingConf) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return c.PingWithContext(ctx, host, hf, conf)
}

// Once sends a single ping and returns it
func Once(host string, conf *PingConf) (*ping.Ping, error) {
	return DefaultConn().Once(host, conf)
}

// Once sends a single ping and returns it
func (c *Conn) Once(host string, conf *PingConf) (*ping.Ping, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if conf == nil {
		conf = DefaultPingConf()
	}
	conf.Count = 1
	var p *ping.Ping
	hf := func(ctx context.Context, rp *ping.Ping, err error) {
		p = rp
	}
	err := c.PingWithContext(ctx, host, hf, conf)
	return p, err
}

// PingWithContext starts a ping. See https://godoc.org/github.com/TrilliumIT/go-multiping/pinger#Conn.PingWithContext
func PingWithContext(ctx context.Context, host string, hf HandleFunc, conf *PingConf) error {
	return DefaultConn().PingWithContext(ctx, host, hf, conf)
}

// PingWithContext starts a new ping to host calling hf every time a reply is recieved or a packet times out
//
// Timed out packets will return an error of context.DeadlineExceeded to hf
//
// Ping will send up to count pings, or unlimited if count is 0
//
// Ping will send a ping each interval. If interval is 0 ping will flood ping, sending a new packet as soon
// as the previous one is returned or timed out
//
// Packets will be considered timed out after timeout. 0 will disable timeouts
//
// With a disabled, or very long timeout and a short interval the packet sequence may rollover and try to reuse
// a packet sequence before the last packet sent with that sequence is recieved. If this happens, hf will recieve
// ErrSeqWrapped
//
// GoRoutines may leake if context is never canceled
func (c *Conn) PingWithContext(ctx context.Context, host string, hf HandleFunc, conf *PingConf) error {
	conf = conf.validate()

	pktWg := sync.WaitGroup{}
	var tick ticker.Ticker
	if conf.Interval > 0 {
		tick = ticker.NewIntervalTicker(conf.Interval, conf.RandDelay)
	} else {
		tick = ticker.NewFloodTicker(pktWg.Wait)
	}
	tickCtx, tickCancel := context.WithCancel(ctx)
	defer tickCancel()
	go tick.Run(tickCtx)
	tick.Ready()

	return c.pingWithTicker(ctx, tick, &pktWg, host, hf, conf)
}

// NewPinger creates a new pinger for manually sending pings. See https://godoc.org/github.com/TrilliumIT/go-multiping/pinger#Conn.NewPinger
func NewPinger(ctx context.Context, host string, hf HandleFunc, conf *PingConf) (run func() error, send func()) {
	return DefaultConn().NewPinger(ctx, host, hf, conf)
}

// NewPinger returns two functions, a run() function to run the listener, and a send() function to send pings.
//
// run() will block until ctx is canceled or an error occors.
//
// calling send() before run() will block.
//
// Interval, and RandDelay are ignored and pings are only sent when the send() function is called.
func (c *Conn) NewPinger(ctx context.Context, host string, hf HandleFunc, conf *PingConf) (run func() error, send func()) {
	conf = conf.validate()

	pktWg := sync.WaitGroup{}
	tick := ticker.NewManualTicker()
	tickCtx, tickCancel := context.WithCancel(ctx)

	return func() error {
		defer tickCancel()
		go tick.Run(tickCtx)
		return c.pingWithTicker(ctx, tick, &pktWg, host, hf, conf)
	}, tick.Tick
}
