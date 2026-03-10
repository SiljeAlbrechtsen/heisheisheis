package setup

import (
	"Project/Network/bcast"
	"Project/Network/localip"
	"Project/Network/peers"
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

// Tar inn vår worldview og kanalen vi skal sende på, og legger periodisk worldview inn på tx-kanalen
func TransmitWorldviewPeriodically(worldviewTx chan<- Worldview, worldviewToNetworkCh <-chan Worldview) {
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

// Tar inn worldviewen vi mottar på Rx og setter den på kanalen som sender til worldview
func ForwardWorldviewFromNetwork(worldviewRx <-chan Worldview, networkToWorldviewCh chan<- Worldview) {
	for {
		wv := <-worldviewRx
		networkToWorldviewCh <- wv
	}
}

func GetNodeID() string {
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

func StartPeerDiscovery(id string) <-chan peers.PeerUpdate {
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

func SetupWorldviewNetwork() (chan<- Worldview, <-chan Worldview) {
	// We make channels for sending and receiving our custom data types
	worldviewTx := make(chan Worldview)
	worldviewRx := make(chan Worldview)

	// And start the transmitter/receiver pair on some port
	go bcast.Transmitter(10002, worldviewTx)
	go bcast.Receiver(10002, worldviewRx)

	return worldviewTx, worldviewRx
}
