### Efficiently Ping lots of hosts in go

Multiping is a library for efficiently pinging lots of hosts. No matter how many
hosts you start pinging a single [Socket](https://godoc.org/github.com/TrilliumIT/go-multiping/ping#Socket) will only open one raw socket listner (two if you are pinging both IPv4 and IPv6 hosts).

The socket will only be listening as long as you have an active ping running.

Note that this library only performs ICMP based pings, which means it must be
run as root or have the appropriate capabilities set. To run tests, or use `go
run` try `go test -exec sudo` or `go run -exec sudo`.

See the [godoc](https://godoc.org/github.com/TrilliumIT/go-multiping/ping) for
more details or look at the example ping command implementation in
[ping.go](cmd/ping/ping.go).

[![Build Status](https://travis-ci.org/TrilliumIT/go-multiping.svg?branch=master)](https://travis-ci.org/TrilliumIT/go-multiping)

[![Coverage Status](https://coveralls.io/repos/github/TrilliumIT/go-multiping/badge.svg?branch=master)](https://coveralls.io/github/TrilliumIT/go-multiping?branch=master)
