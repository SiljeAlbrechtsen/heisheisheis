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

// TransmitWorldviewPeriodically sender vår worldview periodisk på nettverket.
// Sender alltid siste versjon hvert 100ms.
func TransmitWorldviewPeriodically(worldviewTx chan<- wv.Worldview, worldviewToNetworkCh <-chan wv.Worldview) {
	currentMsg := <-worldviewToNetworkCh

	for {
		select {
		case newMsg := <-worldviewToNetworkCh:
			currentMsg = newMsg
		case <-time.After(100 * time.Millisecond):
			worldviewTx <- currentMsg
		}
	}
}

// ForwardWorldviewFromNetwork videresender mottatte worldviews fra nettverket til worldview-goroutinen.
// Sender ikke-blokkerende til init-kanalen (kun relevant ved oppstart).
func ForwardWorldviewFromNetwork(worldviewRx <-chan wv.Worldview, networkToWorldviewCh chan<- wv.Worldview, networkToInitCh chan<- wv.Worldview) {
	for {
		received := <-worldviewRx
		select {
		case networkToInitCh <- received:
		default:
		}
		networkToWorldviewCh <- received
	}
}

// GetNodeID henter node-ID fra kommandolinjeflagg (-id), eller genererer en unik ID
// basert på lokal IP og prosess-ID.
func GetNodeID() string {
	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

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

// StartPeerDiscovery starter peer-oppdagelse og returnerer kanaler for nye og tapte peers.
func StartPeerDiscovery(id string) (<-chan string, <-chan string) {
	peerUpdateCh := make(chan peers.PeerUpdate, 1)
	newPeerIdCh := make(chan string, 1)
	lostPeerIdCh := make(chan string, 1)

	peerTxEnable := make(chan bool)
	go peers.Transmitter(10001, id, peerTxEnable)
	go peers.Receiver(10001, peerUpdateCh)

	go func() {
		for update := range peerUpdateCh {
			fmt.Printf("Peer update: peers=%q new=%q lost=%q\n", update.Peers, update.New, update.Lost)
			if update.New != "" {
				newPeerIdCh <- update.New
			}
			for _, lostId := range update.Lost {
				lostPeerIdCh <- lostId
			}
		}
	}()

	return newPeerIdCh, lostPeerIdCh
}

// SetupWorldviewNetwork oppretter kanaler og starter broadcast-sender/-mottaker for worldview.
func SetupWorldviewNetwork() (chan<- wv.Worldview, <-chan wv.Worldview) {
	worldviewTx := make(chan wv.Worldview)
	worldviewRx := make(chan wv.Worldview)

	go bcast.Transmitter(10002, worldviewTx)
	go bcast.Receiver(10002, worldviewRx)

	return worldviewTx, worldviewRx
}
