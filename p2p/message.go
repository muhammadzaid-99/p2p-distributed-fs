package p2p

import "net"

// Message holds any arbitrary data being sent over
// each transport between two nodes in the network.
type Message struct {
	From    net.Addr
	Payload []byte
}
