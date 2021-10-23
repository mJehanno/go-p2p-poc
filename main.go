package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

func handleStream(stream network.Stream) {
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	go readData(rw, &stream)
	go writeData(rw)
}

func readData(rw *bufio.ReadWriter, s *network.Stream) {

	for {
		stream := *s
		str, err := rw.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Println(err)
			stream.Close()
			fmt.Println("Error reading from buffer")
			//panic(err)
		}
		f, _ := os.OpenFile("readEvents.json", os.O_APPEND|os.O_CREATE, 0644)
		f.WriteString(str)
		f.Close()
		if str == "" {
			return
		}

		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}

	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {

		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}

		_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
		if err != nil {
			fmt.Println("Error writing to buffer")
			panic(err)
		}
		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer")
			panic(err)
		}
	}
}

func main() {
	// create a background context (i.e. one that never cancels)
	ctx := context.Background()

	// start a libp2p node with default settings
	node, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		panic(err)
	}
	node.SetStreamHandler("party", handleStream)
	// print the node's listening addresses
	fmt.Println("Listen addresses:", node.Addrs())

	ser := mdns.NewMdnsService(node, "partyer")
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)
	ser.RegisterNotifee(n)
	peer := <-n.PeerChan
	fmt.Println("peer found at : ", peer)
	if !reflect.DeepEqual(peer.Addrs, node.Addrs()) {
		if err := node.Connect(ctx, peer); err != nil {
			fmt.Println("Connection failed:", err)
		}

		content, err := ioutil.ReadFile("events.json")
		if err != nil {
			fmt.Println("error while reading file : ", err)
		}
		fmt.Println("file content : ", string(content))
		fmt.Println("peer : ", peer.ID)
		stream, err := node.NewStream(ctx, peer.ID, "party")
		if err != nil {
			fmt.Println("error while creating stream : ", err)
		}
		_, er := stream.Write(content)
		if er != nil {
			fmt.Println("error while writing to stream : ", er)
		}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Received signal, shutting down...")

	// shut the node down
	if err := node.Close(); err != nil {
		panic(err)
	}

}

type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

//interface to be called when new  peer is found
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.PeerChan <- pi

}
