package setup

import (
	"Project/Network/bcast"
	"Project/Network/localip"
	"Project/Network/peers"
	wv "Project/worldview"
	"flag"
	"fmt"
	"os"
	"time"
)

//////// TESTING AV NETWORK PACKAGE //////////

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//  will be received as zero-values.

// Tar inn vår worldview og kanalen vi skal sende på, og legger periodisk worldview inn på tx-kanalen
func TransmitWorldviewPeriodically(worldviewTx chan<- wv.Worldview, worldviewToNetworkCh <-chan wv.Worldview) {
	WorldviewMsg := <-worldviewToNetworkCh

	for {
		select {
		// Hvis worldview endres
		case newMsg := <-worldviewToNetworkCh:
			WorldviewMsg = newMsg

		// sender worldview etter 1 sek
		case <-time.After(100 * time.Millisecond):
			worldviewTx <- WorldviewMsg
		}
	}
}

// Tar inn worldviewen vi mottar på Rx og setter den på kanalen som sender til worldview
func ForwardWorldviewFromNetwork(worldviewRx <-chan wv.Worldview, networkToWorldviewCh chan<- wv.Worldview, networkToInitCh chan<- wv.Worldview) {
	for {
		wv := <-worldviewRx
		fmt.Println("Worldview fra: ", wv.IdElevator)
		fmt.Println("Hallorders: ", wv.HallOrders)
		fmt.Println("State: ", wv.State)
		fmt.Println("allCaborders: ", wv.AllCabOrders)

		networkToInitCh <- wv
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

// Endret: returnerer ikke lenger peerUpdateCh for å unngå to lesere på samme kanal
func StartPeerDiscovery(id string) (<-chan string, <-chan string) {
	peerUpdateCh := make(chan peers.PeerUpdate)
	newPeerIdCh := make(chan string)
	lostPeerIdCh := make(chan string)

	peerTxEnable := make(chan bool)
	go peers.Transmitter(10001, id, peerTxEnable)
	go peers.Receiver(10001, peerUpdateCh)

	go func() {
		for update := range peerUpdateCh {

			// Flyttet hit fra main.go
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers: %q\n", update.Peers)
			fmt.Printf("  New:   %q\n", update.New)
			fmt.Printf("  Lost:  %q\n", update.Lost)

			if update.New != "" {
				select {
				case newPeerIdCh <- update.New: // endret: non-blocking
				default:
				}
			}

			for _, lostId := range update.Lost {
				lostPeerIdCh <- lostId
			}
		}
	}()

	return newPeerIdCh, lostPeerIdCh
}

func SetupWorldviewNetwork() (chan<- wv.Worldview, <-chan wv.Worldview) {
	// We make channels for sending and receiving our custom data types
	worldviewTx := make(chan wv.Worldview)
	worldviewRx := make(chan wv.Worldview)

	// And start the transmitter/receiver pair on some port
	go bcast.Transmitter(10002, worldviewTx)
	go bcast.Receiver(10002, worldviewRx)

	return worldviewTx, worldviewRx
}
