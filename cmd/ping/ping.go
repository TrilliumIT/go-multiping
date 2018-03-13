package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	//"runtime/pprof"
	"sync"
	"time"

	"github.com/clinta/go-multiping/packet"
	"github.com/clinta/go-multiping/pinger"
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

	var recieved, dropped uint64
	var clock sync.Mutex

	onReply := func(pkt *packet.Packet) {
		clock.Lock()
		defer clock.Unlock()
		recieved++
		fmt.Printf("%v bytes from %v rtt: %v ttl: %v seq: %v id: %v\n", pkt.Len, pkt.Src.String(), pkt.RTT, pkt.TTL, pkt.Seq, pkt.ID)
		fmt.Printf("%v recieved, %v dropped\n", recieved, dropped)
	}

	onTimeout := func(pkt *packet.SentPacket) {
		clock.Lock()
		defer clock.Unlock()
		dropped++
		fmt.Printf("Packet timed out from %v seq: %v id: %v\n", pkt.Dst.String(), pkt.Seq, pkt.ID)
		fmt.Printf("%v recieved, %v dropped\n", recieved, dropped)
	}

	var wg sync.WaitGroup
	var stops []func()
	for _, h := range flag.Args() {
		d := pinger.NewDst(h, *interval, *timeout, *count)
		d.SetOnReply(onReply)
		d.SetOnTimeout(onTimeout)
		if *reResolve {
			d.SetOnResolveError(func(p *packet.SentPacket, err error) {
				fmt.Printf("Error resolving %v: %v\n", h, err)
			})
		}
		if *randDelay {
			d.EnableRandDelay()
		}
		d.SetOnSendError(func(pkt *packet.SentPacket, err error) {
			fmt.Printf("Error sending to %v: %v\n", h, err)
		})

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := d.Run()
			if err != nil {
				panic(err)
			}
		}()
		stops = append(stops, d.Stop)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		//pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		for _, s := range stops {
			s()
		}
	}()

	wg.Wait()

}
