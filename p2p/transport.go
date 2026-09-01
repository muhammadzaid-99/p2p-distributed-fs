package p2p

import "net"

// Peer represents the remote node.
type Peer interface {
	RemoteAddr() net.Addr
	Close() error
}

// Transport handles communication between the network nodes which
// can be TCP, UDP, websockets, etc.
type Transport interface {
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
