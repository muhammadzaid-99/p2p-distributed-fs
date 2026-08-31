package p2p

import (
	"errors"
)

var ErrInvalidHandshake = errors.New("invalid handshake: handshake between local and remote node could not be established")

type HandshakeFunc func(Peer) error

func NOPHandshakeFunc(Peer) error {
	return nil
}
