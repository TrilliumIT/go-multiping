package main

import (
	"fmt"
	"golang.org/x/net/icmp"
	"time"
)

func main() {
	c, err := icmp.ListenPacket("ip4:icmp", "127.0.0.5")
	if err != nil {
		panic(err)
	}
	go func() {
		time.Sleep(time.Second)
		err := c.IPv4PacketConn().Close()
		if err != nil {
			panic(err)
		}
	}()
	b := make([]byte, 2000)
	i, _, a, err := c.IPv4PacketConn().ReadFrom(b)
	fmt.Printf("i: %v, a: %v, err: %v", i, a, err)
	fmt.Println("success")
}
