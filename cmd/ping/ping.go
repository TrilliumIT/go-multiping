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
	"sync/atomic"
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
	flood := flag.Bool("f", false, "")
	quiet := flag.Bool("q", false, "")
	flag.Usage = func() {
		fmt.Print(usage)
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	var recieved, dropped, errored int64

	handle := func(pkt *ping.Ping, err error) {
		if err == nil {
			atomic.AddInt64(&recieved, 1)
			if !*quiet {
				fmt.Printf("%v bytes from %v rtt: %v ttl: %v seq: %v id: %v count: %v\n", pkt.Len, pkt.Src.String(), pkt.RTT(), pkt.TTL, pkt.Seq, pkt.ID, pkt.Count)
			}
			return
		}

		if err == ping.ErrTimedOut {
			atomic.AddInt64(&dropped, 1)
			if !*quiet {
				fmt.Printf("Packet timed out from %v seq: %v id: %v count: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID, pkt.Count)
			}
			return
		}

		atomic.AddInt64(&errored, 1)
		if !*quiet {
			fmt.Printf("Packet errored from %v seq: %v id: %v count: %v err: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID, pkt.Count, err)
		}
	}

	ping.DefaultSocket().SetWorkers(*workers)

	ctx, cancel := context.WithCancel(context.Background())

	sends := []func(){}
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

	var wg sync.WaitGroup
	for _, host := range flag.Args() {
		switch {
		case *manual:
			fmt.Println("Setting up manual")
			hc := ping.NewHostConn(host, *reResolve, handle, *timeout)
			sends = append(sends, hc.SendPing)
			closes = append(closes, hc.Close)
		case *flood:
			wg.Add(1)
			go func(host string) {
				err := ping.HostFlood(ctx, host, *reResolve, handle, *count, *timeout)
				if err != nil {
					panic(err)
				}
				wg.Done()
			}(host)
		default:
			wg.Add(1)
			go func(host string) {
				err := ping.HostInterval(ctx, host, *reResolve, handle, *count, *interval, *timeout)
				if err != nil {
					panic(err)
				}
				wg.Done()
			}(host)
		}
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
	fmt.Printf("%v recieved, %v dropped, %v errored\n", atomic.LoadInt64(&recieved), atomic.LoadInt64(&dropped), atomic.LoadInt64(&errored))
}
