package p2p

// Peer represents the remote node.
type Peer interface {
	Close() error
}

// Transport handles communication between the network nodes which
// can be TCP, UDP, websockets, etc.
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
}
