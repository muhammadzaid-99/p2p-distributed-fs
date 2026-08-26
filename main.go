package main

import (
	"log"

	"github.com/muhammadzaid-99/distfs/p2p"
)

func main() {
	tr := p2p.NewTCPTransport(":3030")
	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}
