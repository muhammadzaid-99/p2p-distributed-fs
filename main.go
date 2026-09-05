package main

import (
	"bytes"
	"fmt"
	"io"
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
		EncKey:            newEncryptionKey(),
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
	s3 := makeServer(":5000", ":4000", ":3000")

	go func() { log.Fatal(s1.Start()) }()
	time.Sleep(1 * time.Second)

	go s2.Start()
	time.Sleep(1 * time.Second)

	go s3.Start()
	time.Sleep(1 * time.Second)

	for i := range 1 {
		key := fmt.Sprintf("image_%d", i)
		data := bytes.NewReader([]byte("some random data bytes!"))
		if err := s2.Store(key, data); err != nil {
			fmt.Println("Store error", err)
		}

		if err := s2.store.Delete(key); err != nil {
			log.Fatal(err)
		}

		r, err := s2.Get(key)
		if err != nil {
			log.Fatal(err)
		}
		b, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(b))
	}
}
