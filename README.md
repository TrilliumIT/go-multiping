### Efficiently Ping lots of hosts in go

Multiping is a library for efficiently pinging lots of hosts. No matter how many
hosts you start pinging a single [Conn](https://godoc.org/github.com/TrilliumIT/go-multiping/pinger#Conn) will only open one raw socket listner (two if you are pinging both IPv4 and IPv6 hosts).

The socket will only be listening as long as you have an active ping running.

Note that this library only performs ICMP based pings, which means it must be
run as root or have the appropriate capabilities set. To run tests, or use `go
run` try `go test -exec sudo` or `go run -exec sudo`.

See the [godoc](https://godoc.org/github.com/TrilliumIT/go-multiping/pinger) for
more details.

Quick Start.
```go
package main

import (
	"context"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
	"github.com/TrilliumIT/go-multiping/pinger"
)

func handle(ctx context.Context, pkt *ping.Ping, err error) {
	if err == nil {
		fmt.Printf("%v bytes from %v rtt: %v ttl: %v seq: %v id: %v\n", pkt.Len, pkt.Src.String(), pkt.RTT(), pkt.TTL, pkt.Seq, pkt.ID)
		return
	}
	if err == pinger.ErrTimedOut {
		fmt.Printf("Packet timed out from %v seq: %v id: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID)
		return
	}
	fmt.Printf("Packet errored from %v seq: %v id: %v err: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID, err)
}

func main() {
	conf := pinger.DefaultPingConf()
	conf.Count = 1
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	for _, h := range flag.Args() {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()
			err := pinger.PingWithContext(ctx, h, handle, conf)
			if err != nil {
				panic(err)
			}
		}(h)
	}
	wg.Wait()
}
```

A more detailed example is included in [ping.go](cmd/ping/ping.go)

[![Build Status](https://travis-ci.org/TrilliumIT/go-multiping.svg?branch=master)](https://travis-ci.org/TrilliumIT/go-multiping)

[![Coverage Status](https://coveralls.io/repos/github/TrilliumIT/go-multiping/badge.svg?branch=master)](https://coveralls.io/github/TrilliumIT/go-multiping?branch=master)
