package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/TrilliumIT/go-multiping/ping"
	"github.com/TrilliumIT/go-multiping/pinger"
)

var usage = `
Usage:

    ping [-c count] [-i interval] [-t timeout] [-r] [-d] host host2 host3

Examples:

    # ping google continuously
    ping www.google.com

    # ping google 5 times
    ping -c 5 www.google.com

    # ping google 5 times at 500ms intervals
    ping -c 5 -i 500ms www.google.com

    # ping google with a 500ms timeout re-resolving dns every ping
    ping -t 500ms -r www.google.com
`

func main() {
	timeout := flag.Duration("t", time.Second, "")
	interval := flag.Duration("i", time.Second, "")
	count := flag.Int("c", 0, "")
	reResolve := flag.Bool("r", false, "")
	randDelay := flag.Bool("d", false, "")
	flag.Usage = func() {
		fmt.Print(usage)
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	var recieved, dropped, errored uint64
	var clock sync.Mutex

	callBack := func(pkt *ping.Ping, err error) {
		clock.Lock()
		if err != nil {
			errored++
		} else {
			if pkt.IsRecieved() && !pkt.IsTimedOut() {
				recieved++
			} else {
				dropped++
			}
		}
		clock.Unlock()
		if err != nil {
			fmt.Printf("Packet errored from %v seq: %v id: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID)
		}
		if pkt.IsRecieved() {
			fmt.Printf("%v bytes from %v rtt: %v ttl: %v seq: %v id: %v\n", pkt.Len, pkt.Src.String(), pkt.RTT(), pkt.TTL, pkt.Seq, pkt.ID)
		}
		if pkt.IsTimedOut() {
			fmt.Printf("Packet timed out from %v seq: %v id: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID)
		}
		fmt.Printf("%v recieved, %v dropped\n", recieved, dropped)
	}

	pinger.DefaultConn().SetWorkers(4)
	conf := pinger.DefaultPingConf()
	conf.RetryOnResolveError = *reResolve
	if *reResolve {
		conf.ReResolveEvery = 1
	}
	conf.Interval = *interval
	conf.Timeout = *timeout
	conf.Count = *count
	conf.RandDelay = *randDelay
	conf.RetryOnSendError = true

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	for _, h := range flag.Args() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := pinger.PingWithContext(ctx, h, callBack, conf)
			if err != nil {
				panic(err)
			}
		}()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
		go func() {
			time.Sleep(30 * time.Second)
			_ = pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			panic("die")
		}()
	}()

	wg.Wait()

}
