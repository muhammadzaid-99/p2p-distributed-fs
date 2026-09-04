package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return nil
	}

	// Not decoded in case of stream
	if peekBuf[0] == StreamIncoming {
		msg.Stream = true
		return nil
	}

	buf := make([]byte, 10240)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}

	// fmt.Println(":::Default Decoder Start:::")
	// fmt.Println(buf[:n])
	// fmt.Println(":::Default Decoder End:::")

	msg.Payload = buf[:n]

	return nil
}
