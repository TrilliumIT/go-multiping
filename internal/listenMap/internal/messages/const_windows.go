// see https://github.com/golang/net/blob/master/ipv4/control_windows.go#L14

package messages

import (
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	v4AddLen = ipv4.HeaderLen
	v6AddLen = ipv6.HeaderLen
)
