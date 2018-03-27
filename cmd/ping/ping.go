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

	"github.com/pkg/profile"
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
	defer profile.Start().Stop()
	timeout := flag.Duration("t", time.Second, "")
	interval := flag.Duration("i", time.Second, "")
	count := flag.Int("c", 0, "")
	workers := flag.Int("w", 1, "")
	reResolve := flag.Int("r", 1, "")
	//randDelay := flag.Bool("d", false, "")
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

	handle := func(pkt *ping.Ping, err error) {
		clock.Lock()
		defer clock.Unlock()
		if err == nil {
			recieved++
			fmt.Printf("%v bytes from %v rtt: %v ttl: %v seq: %v id: %v count: %v\n", pkt.Len, pkt.Src.String(), pkt.RTT(), pkt.TTL, pkt.Seq, pkt.ID, pkt.Count)
			return
		}

		dropped++
		if err == ping.ErrTimedOut {
			fmt.Printf("Packet timed out from %v seq: %v id: %v count: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID, pkt.Count)
			return
		}

		fmt.Printf("Packet errored from %v seq: %v id: %v err: %v count: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID, pkt.Count, err)
	}
	if *quiet {
		handle = func(pkt *ping.Ping, err error) {
			clock.Lock()
			if err == nil {
				recieved++
				clock.Unlock()
				return
			}
			dropped++
			clock.Unlock()
		}
	}

	ping.DefaultSocket().SetWorkers(*workers)

	ctx, cancel := context.WithCancel(context.Background())

	runs := []func() error{}
	sends := []func() int{}
	closes := []func() error{}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
		go func() {
			time.Sleep(10 * time.Second)
			err := pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			panic(err)
		}()
		for _, cl := range closes {
			if err := cl(); err != nil {
				panic(err)
			}
		}
	}()

	for _, host := range flag.Args() {
		switch {
		case *manual:
			fmt.Println("Setting up manual")
			hc := ping.NewHostConn(host, *reResolve, handle, *timeout)
			sends = append(sends, hc.SendPing)
			closes = append(closes, hc.Close)
		case *interval == 0:
			runs = append(runs, func() error { return ping.HostFlood(ctx, host, *reResolve, handle, *count, *timeout) })
		default:
			runs = append(runs, func() error { return ping.HostInterval(ctx, host, *reResolve, handle, *count, *interval, *timeout) })
		}
	}

	var wg sync.WaitGroup
	for _, run := range runs {
		wg.Add(1)
		go func(r func() error) {
			if err := r(); err != nil {
				panic(err)
			}
			wg.Done()
		}(run)
	}

	if *manual {
		fmt.Println("Press 'Enter' to send pings...")
		for {
			_, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
			if err != nil {
				panic(err)
			}
			for _, s := range sends {
				s()
			}
		}
	}

	wg.Wait()
	clock.Lock()
	fmt.Printf("%v recieved, %v dropped, %v errored\n", recieved, dropped, errored)
	clock.Unlock()
}
