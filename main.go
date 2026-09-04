package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/muhammadzaid-99/distfs/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       strings.TrimPrefix(listenAddr, ":") + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}
	s := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer

	return s
}

func main() {
	s1 := makeServer(":3000")
	s2 := makeServer(":4000", ":3000")

	go func() { log.Fatal(s1.Start()) }()
	time.Sleep(1 * time.Second)

	go s2.Start()
	time.Sleep(1 * time.Second)

	data := bytes.NewReader([]byte("some random data bytes!"))

	for i := range 10 {
		if err := s2.Store(fmt.Sprintf("myprivatedata_%d", i+1), data); err != nil {
			fmt.Println("Store error", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// r, err := s2.Get("myprivatedata")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// b, err := io.ReadAll(r)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(string(b))

	select {}
}
