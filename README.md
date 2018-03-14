### Efficiently Ping lots of hosts in go

Multiping is a library for efficiently pinging lots of hosts. No matter how many
hosts you start pinging a pinger will only open one raw socket listner (two
if you are pinging both IPv4 and IPv6 hosts).

The socket will only be listening as long as you have an active ping running.
Pingers are thread safe and you can add and remove destinations as needed.

See the [godoc](https://godoc.org/github.com/TrilliumIT/go-multiping/pinger) for
more details.

Quick Start.
```go
import (
  "sync"
  "time"

	"github.com/TrilliumIT/go-multiping/packet"
	"github.com/TrilliumIT/go-multiping/pinger"
)

func main() {
	onReply := func(pkt *packet.Packet) {
		fmt.Printf("%v bytes from %v rtt: %v ttl: %v seq: %v id: %v\n", pkt.Len, pkt.Src.String(), pkt.RTT, pkt.TTL, pkt.Seq, pkt.ID)
	}

	onTimeout := func(pkt *packet.Packet) {
		fmt.Printf("Packet timed out from %v seq: %v id: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID)
	}

	var wg sync.WaitGroup
	for _, h := range flag.Args() {
		d := pinger.NewDst(h, 4, time.Second, time.Second)
		wg.Add(1)
		go func() {
			d.Run()
			wg.Done()
		}
	}
	wg.Wait()
}
```

A more detailed example is included in [ping.go](cmd/ping/ping.go)

[![Build Status](https://travis-ci.org/TrilliumIT/go-multiping.svg?branch=master)](https://travis-ci.org/TrilliumIT/go-multiping)

[![Coverage Status](https://coveralls.io/repos/github/TrilliumIT/go-multiping/badge.svg?branch=master)](https://coveralls.io/github/TrilliumIT/go-multiping?branch=master)
