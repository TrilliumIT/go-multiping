package socket

type Socket struct {
	v4m  ip4IdMap
	v4To timeoutMap
	v4c  conn

	v6m  ip6IdMap
	v6To timeoutMap
	v6c  conn
}
