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

    ping [-c count] [-i interval] [-t timeout] [-w workers] [-b buffersize] [-r] [-d] host host2 host3

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
	workers := flag.Int("w", 0, "")
	buffer := flag.Int("b", 0, "")
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

	handle := func(ctx context.Context, pkt *ping.Ping, err error) {
		clock.Lock()
		defer clock.Unlock()
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err == nil {
			recieved++
			fmt.Printf("%v bytes from %v rtt: %v ttl: %v seq: %v id: %v\n", pkt.Len, pkt.Src.String(), pkt.RTT(), pkt.TTL, pkt.Seq, pkt.ID)
			return
		}

		dropped++
		if err == pinger.ErrTimedOut {
			fmt.Printf("Packet timed out from %v seq: %v id: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID)
			return
		}

		fmt.Printf("Packet errored from %v seq: %v id: %v err: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID, err)
	}

	pinger.DefaultConn().SetWorkers(*workers)
	pinger.DefaultConn().SetBuffer(*buffer)
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
		go func(h string) {
			defer wg.Done()
			err := pinger.PingWithContext(ctx, h, handle, conf)
			if err != nil {
				panic(err)
			}
		}(h)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
		go func() {
			time.Sleep(30 * time.Second)
			err := pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			panic(err)
		}()
	}()

	wg.Wait()
	clock.Lock()
	fmt.Printf("%v recieved, %v dropped, %v errored\n", recieved, dropped, errored)
	clock.Unlock()

}
