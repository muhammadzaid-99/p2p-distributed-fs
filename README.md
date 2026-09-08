# distfs

A peer-to-peer distributed file store written in Go.

You hand a node a key and a stream of bytes. The node writes those bytes to its own disk
and, at the same time, pushes an encrypted copy to every peer it is connected to. Later
you can ask any node for that key. If it has the file locally it serves it from disk. If
it does not, it asks the network, pulls the file back over TCP, decrypts it, and writes
it down so the next request is local.

There is no coordinator, no name server and no metadata database. Every node runs the same
code, and the whole thing is built on the standard library, with the network layer, the
storage layer and the crypto layer kept separate enough that any one of them can be
swapped out.

## What it does

**Content addressable storage.** Files are never stored under the name you gave them. A
key is hashed and the hash decides where the file lives on disk, so the layout is
deterministic, evenly spread, and free of anything the filesystem might object to.

**Replication on write.** Writing to one node writes to all of its peers. There is no
separate sync step; a single pass over your input feeds both the local file and the network.

**Retrieval across the network.** A node that is missing a file goes and finds it.
Retrieval reuses the same connections and the same framing as writes.

**End to end streaming.** Files move through the system as `io.Reader` and `io.Writer`
pipelines. Bytes go from a socket straight into a file handle, or from a file handle
straight into a socket.

**Encryption in transit, and at rest on peers.** Every node has its own key. Files leave
the node encrypted, and the peers holding the replicas hold ciphertext they cannot read.

**Pluggable pieces.** The transport, the message decoder, the handshake and the on-disk
path scheme are all interfaces with a default implementation behind them.

## How it works

### Content addressable path layout

A store that dumps every file into one directory runs into two problems. Keys can contain
characters the filesystem will not accept, and directories holding very large numbers of
entries get slow to walk.

`CASPathTransformFunc` in `store.go` takes the SHA-1 of the key, hex encodes it into 40
characters, and cuts that into eight blocks of five. The blocks become nested directory
names and the full hash becomes the filename:

`3000_network/<node-id>/fb994/fe8d2/dee46/dc218/ba3b6/b7d66/8b3cd/ba598/fb994fe8d2dee46...`

The result is a tree that fans out evenly no matter what the input keys look like, with
filenames that are always safe to write. It also makes deletion cheap and exact. `Delete`
takes the first path segment and removes that whole subtree, so a file and the directories
created for it go away together.

The scheme is a function, not a hardcoded rule. `Store` takes a `PathTransformFunc`, and
`DefaultPathTransformFunc`, which uses the key verbatim, exists mainly to keep tests
readable.

### Node identity and per-owner namespaces

A node's disk holds two different kinds of file: the ones it stored itself, and the replicas
it is holding for other nodes. If both went into the same tree, two nodes that happened to
use the key `avatar` would collide, and neither could tell whose file it was looking at.

Each server generates a random 32 byte identifier at startup (`generateID` in `crypto.go`),
and every path is built as `Root/<id>/<hashed path>`. The important part is that the
identifier travels with the data. `MessageStoreFile` carries the originator's ID, and the
receiving node writes the replica under that ID rather than under its own. A replica
therefore sits in exactly the place its owner would look for it, and a later request naming
the same ID lands on the same path.

This is what lets the network work without a lookup table. The owner's ID plus the hashed
key is the address, and that address means the same thing on every machine.

### One connection, two kinds of traffic

This is the part of the design most worth explaining. A single TCP connection between two
nodes has to carry two very different things: small control messages such as "I am about to
store a file called X", and file payloads that may be arbitrarily large. A reader that just
calls `gob.Decode` in a loop cannot handle this, because once a file body starts arriving
there is nothing to tell it that those bytes are not another message.

The solution is a one byte header written ahead of every send. `p2p/message.go` defines the
two values: `0x1` means a gob encoded control message follows, `0x2` means a raw stream
follows. `DefaultDecoder` reads exactly one byte, and when it sees the stream marker it sets
`RPC.Stream` and returns immediately without touching the body.

The payload is deliberately left sitting in the socket. It belongs to the application layer,
not to the decoder.

### Pausing the read loop while the application drains the socket

Leaving the body in the socket creates the obvious follow-on problem. The transport's read
loop runs on its own goroutine per connection, and if it keeps looping it will read file
content and try to interpret it as messages. But the code that actually wants those bytes
lives in the `FileServer`, on a different goroutine, and cannot be called into synchronously.

`TCPPeer` carries a `sync.WaitGroup` for this. When the read loop sees a stream RPC it calls
`wg.Add(1)` and then `wg.Wait()`, which parks that connection's reader. Meanwhile the peer
object, which embeds `net.Conn` and is therefore already an `io.Reader`, is handed to the
application. The handler copies as many bytes as it wants directly off the socket, then calls
`CloseStream()`, which is `wg.Done()`, and the read loop wakes up and carries on.

What this buys is genuine streaming. The transport never buffers a file, never needs to know
how big one is, and never guesses where a payload ends. Ownership of the connection passes to
the application for the length of the transfer and comes back cleanly afterwards.
`handleMessageStoreFile` releases it with a `defer`; the read path releases it once per peer
it read from.

Peers are handed out as an interface (`p2p.Peer`) that embeds `net.Conn`, which is why the
application can treat a remote node as a plain reader or writer with no adapters in between.

### Knowing where a payload ends

Because the transport hands the raw socket to the application, the application has to know
how many bytes belong to the current transfer. Both directions solve this by sending the
length, but at different layers.

On a write, the size rides along in the control message. `Store` counts the bytes it wrote
locally and puts `n + 16` in `MessageStoreFile.Size`, the extra sixteen being the IV that the
encrypting copy prepends. The receiver wraps the peer in an `io.LimitReader` with that size
and copies it to disk.

On a read there is no control message to attach it to, because the reply is the file itself.
`handleMessageGetFile` writes the file size as a little endian `int64` on the wire directly
ahead of the body, and the requester reads that with `binary.Read` before limiting its own
read to that many bytes.

Both routes end at the same place. Nobody ever reads a byte past the end of a transfer, so
the connection is left in a clean state for the next message.

### Writing to disk and to the network in one pass

`Store` needs its input twice, once for the local file and once for every peer, but the input
is an `io.Reader` and can only be consumed once. Rather than read it all into memory first,
it wraps the reader in an `io.TeeReader`, so that writing the local file also fills a buffer
as a side effect. It then encrypts that buffer out to an `io.MultiWriter` built from the
current peer set, so one `copyEncrypt` call fans the ciphertext out to every peer at once.
After broadcasting the control message the sender pauses briefly, so peers have processed it
and are ready to receive the stream that follows.

### Encryption

Files are encrypted with AES-256 in counter mode. CTR was chosen because it turns a block
cipher into a stream cipher: no padding, no block alignment, and any length of input works.
That is exactly what a system built out of `io.Copy` calls needs.

`copyEncrypt` generates a fresh random 16 byte IV per transfer and writes it as the first
bytes of the output, so the IV travels with the data instead of having to be agreed
separately. `copyDecrypt` reads those bytes back off the front and rebuilds the same keystream.
Both are thin wrappers over `copyStream`, which does a chunked read, XOR and write, so
encryption costs one fixed size buffer regardless of file size.

The interesting consequence is where plaintext exists. The originating node writes its own
copy in the clear, but what goes over the wire and lands on the peers is ciphertext, IV
included, and it is stored that way. When a peer serves that file back it does a plain
`io.Copy` of the bytes it already holds and never needs the key. Only the requester, who owns
the key, decrypts, on the way to disk via `WriteDecrypt`. Nodes hold each other's data without
being able to read it, and that falls out of the design rather than needing extra machinery.

### Peer set and bootstrapping

A server is given a list of addresses to dial at startup. `bootstrapNetwork` dials each one in
its own goroutine, so a slow or dead address never holds up the rest, and failures are logged
and dropped rather than aborting startup.

Connections are registered the same way whichever direction they came from. The transport calls
its `OnPeer` callback for every connection it accepts and every connection it dials, and the
server's `OnPeer` puts the peer into a map keyed by remote address, guarded by a `sync.RWMutex`
because connections arrive on many goroutines at once. Incoming RPCs carry the sender's address,
which is how a handler finds the peer to reply on.

The server's main `loop` is a single `select` over the transport's RPC channel and a quit
channel, so all message handling is serialised through one goroutine and shutting the server
down is just closing a channel.

## Running it

```
make build   # builds to bin/fs
make run     # builds and runs the demo
make test    # go test ./... -v
```

`main.go` is a working demo of the whole system. It starts three nodes on ports 3000, 4000 and
5000, wires 4000 to 3000 and 5000 to both, then uses the 4000 node to store a key, delete it
from its own disk, and ask for it again. The file comes back off the network, is decrypted, and
is printed.

Each node keeps its data under a directory named after its port, for example `3000_network`.
Those directories are gitignored.

## Configuration

Configuration happens in Go, through option structs passed to the constructors. There are no
config files or command line flags. The fields that matter:

| Struct | Field | What it controls |
| --- | --- | --- |
| `FileServerOpts` | `StorageRoot` | Root directory for this node's data |
| | `EncKey` | AES key for this node, 32 bytes |
| | `BootstrapNodes` | Addresses dialled on startup |
| | `ID` | Node identity, generated randomly when left empty |
| | `PathTransformFunc` | On-disk layout scheme |
| | `Transport` | Network implementation to use |
| `TCPTransportOpts` | `ListenAddr` | Address to listen on |
| | `HandshakeFunc` | Runs on every new connection before it is used |
| | `Decoder` | Turns bytes on the wire into an `RPC` |
| | `OnPeer` | Called when a connection is established |

`main.go` shows a complete node setup; `server.go` and `p2p/tcp_transport.go` have the full
definitions.

## Project layout

| Path | What is in it |
| --- | --- |
| `main.go` | Node construction and the three node demo |
| `server.go` | `FileServer`: message types, store and get flows, peer registry, main loop |
| `store.go` | Content addressable disk store, path transforms, streaming read and write |
| `crypto.go` | AES-CTR stream encryption, node IDs, key hashing |
| `p2p/` | Transport and Peer interfaces, TCP implementation, wire framing, decoders, handshake |
| `*_test.go` | Unit tests for the store, the crypto helpers and the transport |

## Credit

The architecture of this project follows Anthony GG's
[Distributed File Storage in Go](https://www.youtube.com/watch?v=bymQakvTY40)
series, built along with and extended while working through it.
