package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	mathRand "math/rand"
	"time"
)

var (
	ProtocolID = "/test-request/v0.0.1"
)

type Target struct {
	Url      string `json:"url"`
	Rps      int    `json:"rps"`
	Resident bool   `json:"-"`
}

type Set[K comparable] map[K]struct{}

func (s Set[K]) Add(k K) bool {
	if _, ok := s[k]; !ok {
		s[k] = struct{}{}
		return true
	}
	return false
}

var urls = []string{"https://google.com", "https://facebook.com", "https://2ip.ru"}

var targets = make(Set[Target])

func main() {
	n := 0
	ctx := context.Background()
	r := rand.Reader

	go func() {
		for {
			targets.Add(
				Target{
					Url:      urls[mathRand.Intn(len(urls))] + "/" + string(rune(n)),
					Rps:      mathRand.Intn(100) + 50,
					Resident: true,
				},
			)
			n++
			time.Sleep(time.Second * 10)
		}
	}()

	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", "127.0.0.1", 0))

	host, err := libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		panic(err)
	}
	host.SetStreamHandler(protocol.ID(ProtocolID), func(stream network.Stream) {
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

		go readData(rw.Reader)
		go writeData(rw.Writer)
	})
	peerChan := initMDNS(host, "1234")
	for {
		peer := <-peerChan
		if peer.ID > host.ID() {
			fmt.Println("Found peer:", peer, " id is greater than us , wait for it to connect to us")
			continue
		}
		fmt.Println("Found peer:", peer, ", connecting")

		if err := host.Connect(ctx, peer); err != nil {
			fmt.Println("Connection failed:", err)
			continue
		}

		stream, err := host.NewStream(ctx, peer.ID, protocol.ID(ProtocolID))

		if err != nil {
			fmt.Println("Stream open failed", err)
		} else {
			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

			go writeData(rw.Writer)
			go readData(rw.Reader)
			fmt.Println("Connected to:", peer)
		}
	}
}

func readData(r *bufio.Reader) {
	fmt.Println("starting to read from stream")
	for {
		dumpPart, err := r.ReadBytes('\n')
		if err != nil {
			fmt.Println(err)
		}

		var newTarget Target
		err = json.Unmarshal(dumpPart, &newTarget)
		if err != nil {
			fmt.Println(err)
		}
		targets.Add(newTarget)
		fmt.Println(targets)
	}

}

func writeData(w *bufio.Writer) {
	fmt.Println("starting to write to stream")
	for {
		time.Sleep(time.Second * 5)
		for target, _ := range targets {
			targetDump, err := json.Marshal(target)
			if err != nil {
				fmt.Println(err)
			}
			_, err = w.Write(targetDump)
			if err != nil {
				fmt.Println(err)
			}
			_, err = w.Write([]byte("\n"))
			if err != nil {
				fmt.Println(err)
			}
		}
		err := w.Flush()
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
