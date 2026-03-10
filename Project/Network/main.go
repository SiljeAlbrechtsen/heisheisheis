package main

import (
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

func transmittiWorldviewPeriodically(worldviewTx chan<- Worldview, worldviewToNetworkCh <-chan Worldview) {
	WorldviewMsg := <-worldviewToNetworkCh

	for {
		select {
		// Hvis worldview endres
		case newMsg := <-worldviewToNetworkCh:
			WorldviewMsg = newMsg

		// sender worldview etter 1 sek
		case <-time.After(1 * time.Second):
			worldviewTx <- WorldviewMsg
		}
	}
}

func getNodeID() string {
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
	return id
}

type Worldview struct {
	ID   string
	Iter int
}

func startPeerDiscovery(id string) <-chan peers.PeerUpdate {
	// We make a channel for receiving updates on the id's of the peers that are
	// alive on the network
	peerUpdateCh := make(chan peers.PeerUpdate)

	// We can disable/enable the transmitter after it has been started.
	// This could be used to signal that we are somehow "unavailable".
	peerTxEnable := make(chan bool)
	go peers.Transmitter(10001, id, peerTxEnable)
	go peers.Receiver(10001, peerUpdateCh)

	return peerUpdateCh
}

func setupWorldviewNetwork() (chan<- Worldview, <-chan Worldview) {
	// We make channels for sending and receiving our custom data types
	worldviewTx := make(chan Worldview)
	worldviewRx := make(chan Worldview)

	// And start the transmitter/receiver pair on some port
	go bcast.Transmitter(10002, worldviewTx)
	go bcast.Receiver(10002, worldviewRx)

	return worldviewTx, worldviewRx
}

func 


func main() {

	//__________________________________________________________________
	//----------------  SETTER UNIK ID FOR DENNE NODEN -----------------
	//__________________________________________________________________

	// `go run main.go -id=our_id`
	id := getNodeID()

	//__________________________________________________________________
	//---------------------- PEER DISCOVERY ----------------------------
	//__________________________________________________________________

	peerUpdateCh := startPeerDiscovery(id)

	//__________________________________________________________________
	//------------- STARTER KOMMUNIKASJON MED HEARTBEATS ---------------
	//__________________________________________________________________

	worldviewTx, worldviewRx := setupWorldviewNetwork()

	//__________________________________________________________________
	//----------- SENDER DENNE NODEN SINE HEARTBEATS PERIODISK ---------
	//__________________________________________________________________

	worldviewToNetworkCh := make(chan Worldview)

	go transmittiWorldviewPeriodically(worldviewTx, worldviewToNetworkCh)



	// gorutine som sender fra vårt worldview, erdig formatert, fra worldview til nettverk
	//go elevio.PollFloorSensor(floorCh)

	// The example message. We just send one of these every second.

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

		case a := <-worldviewRx:
			fmt.Printf("Received from %q: %#v\n", id, a)
			//TODO
			// sende mottat wv til worldview, updateWorldview
		}
	}
	
}
