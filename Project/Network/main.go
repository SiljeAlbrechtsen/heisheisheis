package main

import (
	"Driver-go/elevio"
	"Network/network/bcast"
	"Network/network/localip"
	"Network/network/peers"
	"flag"
	"fmt"
	"os"
	"time"
)

//////// TESTING AV NETWORK PACKAGE //////////

//////// TESTING AV NETWORK PACKAGE //////////

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//  will be received as zero-values.

func main() {
	elevio.Init("localhost:15657", 4)

	//__________________________________________________________________
	//----------------  SETTER UNIK ID FOR DENNE NODEN -----------------
	//__________________________________________________________________

	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	// ... or alternatively, we can use the local IP address.
	// (But since we can run multiple programs on the same PC, we also append the
	//  process ID)
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	//__________________________________________________________________
	//---------------------- PEER DISCOVERY ----------------------------
	//__________________________________________________________________

	// We make a channel for receiving updates on the id's of the peers that are
	//  alive on the network
	peerUpdateCh := make(chan peers.PeerUpdate)

	// We can disable/enable the transmitter after it has been started.
	// This could be used to signal that we are somehow "unavailable".
	peerTxEnable := make(chan bool)
	go peers.Transmitter(10001, id, peerTxEnable)
	go peers.Receiver(10001, peerUpdateCh)

	//__________________________________________________________________
	//------------- STARTER KOMMUNIKASJON MED HEARTBEATS ---------------
	//__________________________________________________________________

	// We make channels for sending and receiving our custom data types
	heartbeatTx := make(chan int)
	heartbeatRx := make(chan int)
	// ... and start the transmitter/receiver pair on some port
	// These functi
	//  start multiple transmitters/receivers on the same port.
	go bcast.Transmitter(10002, heartbeatTx)
	go bcast.Receiver(10002, heartbeatRx)

	//__________________________________________________________________
	//----------- SENDER DENNE NODEN SINE HEARTBEATS PERIODISK ---------
	//__________________________________________________________________

	//worldviewCh := make(chan Worldview)

	floorCh := make(chan int)

	go elevio.PollFloorSensor(floorCh)

	// The example message. We just send one of these every second.
	go func() {
		HeartbeatMsg := <-floorCh

		for {
			select {
			// Hvis worldview endres
			case newMsg := <-floorCh:
				HeartbeatMsg = newMsg

			// sender worldview etter 1 sek
			case <-time.After(1 * time.Second):
				heartbeatTx <- HeartbeatMsg
			}
		}
	}()

	//__________________________________________________________________
	//----------------  PRINTER INFORMASJON ----------------------------
	//__________________________________________________________________

	fmt.Println("Started")
	for {
		select {
		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

		case a := <-heartbeatRx:
			fmt.Printf("Received from %q: %#v\n", id, a)
		}
	}
}
