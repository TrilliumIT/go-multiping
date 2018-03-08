package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/clinta/go-multiping/packet"
	"github.com/clinta/go-multiping/pinger"
)

var usage = `
Usage:

    ping [-c count] [-i interval] [-t timeout] host host2 host3

Examples:

    # ping google continuously
    ping www.google.com

    # ping google 5 times
    ping -c 5 www.google.com

    # ping google 5 times at 500ms intervals
    ping -c 5 -i 500ms www.google.com

    # ping google for 10 seconds
    ping -t 10s www.google.com

    # Send a privileged raw ICMP ping
    sudo ping --privileged www.google.com
`

func main() {
	timeout := flag.Duration("t", time.Second, "")
	interval := flag.Duration("i", time.Second, "")
	count := flag.Int("c", 0, "")
	flag.Usage = func() {
		fmt.Printf(usage)
	}
	flag.Parse()

	fmt.Printf("i %v\n", interval.String())
	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	p := pinger.NewPinger()
	f := func(pkt *packet.Packet) {
		fmt.Printf("%v bytes from %v rtt: %v ttl: %v seq: %v id: %v\n", pkt.Len, pkt.Src.String(), pkt.RTT, pkt.TTL, pkt.Seq, pkt.ID)
	}

	var wg sync.WaitGroup
	for _, h := range flag.Args() {
		d, err := p.NewDst(h, *interval, *timeout, *count, f)
		if err != nil {
			panic(err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = d.Run()
			if err != nil {
				panic(err)
			}
		}()
	}

	wg.Wait()

}
