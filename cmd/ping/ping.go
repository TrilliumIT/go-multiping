package main

import (
	"bufio"
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

    ping [-c count] [-i interval] [-t timeout] [-cw workers] [-cb buffersize] [-w workers] [-b buffer] [-r] [-d] [-m] host host2 host3

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
	connWorkers := flag.Int("cw", -1, "")
	connBuffer := flag.Int("cb", 0, "")
	workers := flag.Int("w", -1, "")
	buffer := flag.Int("b", 0, "")
	reResolve := flag.Bool("r", false, "")
	randDelay := flag.Bool("d", false, "")
	manual := flag.Bool("m", false, "")
	quiet := flag.Bool("q", false, "")
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
			if !*quiet {
				fmt.Printf("%v bytes from %v rtt: %v ttl: %v seq: %v id: %v\n", pkt.Len, pkt.Src.String(), pkt.RTT(), pkt.TTL, pkt.Seq, pkt.ID)
			}
			return
		}

		dropped++
		if *quiet {
			return
		}
		if err == pinger.ErrTimedOut {
			fmt.Printf("Packet timed out from %v seq: %v id: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID)
			return
		}

		fmt.Printf("Packet errored from %v seq: %v id: %v err: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID, err)
	}

	pinger.DefaultConn().SetWorkers(*connWorkers)
	pinger.DefaultConn().SetBuffer(*connBuffer)
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
	conf.Workers = *workers
	conf.Buffer = *buffer

	ctx, cancel := context.WithCancel(context.Background())

	sends := []func(){}
	var wg sync.WaitGroup
	for _, h := range flag.Args() {
		host := h
		var run func() error
		var send func()
		if *manual {
			run, send = pinger.NewPinger(ctx, host, handle, conf)
			sends = append(sends, send)
		} else {
			run = func() error { return pinger.PingWithContext(ctx, host, handle, conf) }
		}
		wg.Add(1)
		go func(run func() error) {
			defer wg.Done()
			err := run()
			if err != nil {
				panic(err)
			}
		}(run)
	}

	if *manual {
		go func() {
			fmt.Println("Press 'Enter' to send pings...")
			for {
				bufio.NewReader(os.Stdin).ReadBytes('\n')
				for _, s := range sends {
					s()
				}
			}
		}()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
		go func() {
			time.Sleep(5 * time.Second)
			err := pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			panic(err)
		}()
	}()

	wg.Wait()
	clock.Lock()
	fmt.Printf("%v recieved, %v dropped, %v errored\n", recieved, dropped, errored)
	clock.Unlock()

}
