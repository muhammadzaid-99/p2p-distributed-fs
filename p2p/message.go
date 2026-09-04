package p2p

const (
	MessageIncoming = 0x1
	StreamIncoming  = 0x2
)

// RPC holds any arbitrary data being sent over
// each transport between two nodes in the network.
type RPC struct {
	From    string
	Payload []byte
	Stream  bool
}
