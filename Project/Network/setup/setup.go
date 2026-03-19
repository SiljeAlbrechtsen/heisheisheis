package setup

import (
	"Project/Network/bcast"
	"Project/Network/localip"
	"Project/Network/peers"
	wv "Project/worldview"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ResolveElevatorAddr finds the address of the elevator simulator or hardware.
// It reads ELEVATOR_ADDR from the environment; otherwise it probes the local network on port 15657.
func ResolveElevatorAddr() string {
	if addr := strings.TrimSpace(os.Getenv("ELEVATOR_ADDR")); addr != "" {
		return addr
	}
	candidates := []string{"localhost:15657"}
	if out, err := exec.Command("sh", "-c", "ip route | awk '/default/ {print $3}'").Output(); err == nil {
		ip := strings.TrimSpace(string(out))
		if ip != "" {
			candidates = append(candidates, ip+":15657")
		}
	}
	for _, addr := range candidates {
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return addr
		}
	}
	return candidates[0]
}

// TransmitWorldviewPeriodically sends our worldview periodically over the network.
// It always sends the latest version every 100 ms.
func TransmitWorldviewPeriodically(broadcastTx chan<- wv.Worldview, localWorldviewBroadcastCh <-chan wv.Worldview) {
	currentMsg := <-localWorldviewBroadcastCh

	for {
		select {
		case newMsg := <-localWorldviewBroadcastCh:
			currentMsg = newMsg
		case <-time.After(100 * time.Millisecond):
			broadcastTx <- currentMsg
		}
	}
}

// ForwardWorldviewFromNetwork forwards received worldviews from the network to the worldview goroutine.
// It sends non-blockingly to the init channel, which is only relevant during startup.
func ForwardWorldviewFromNetwork(broadcastRx <-chan wv.Worldview, peerWorldviewCh chan<- wv.Worldview, initialWorldviewCh chan<- wv.Worldview) {
	for {
		received := <-broadcastRx
		select {
		case initialWorldviewCh <- received:
		default:
		}
		peerWorldviewCh <- received
	}
}

// GetNodeID gets the node ID from the command-line flag (-id), or generates a unique ID
// based on the local IP address and process ID.
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

// StartPeerDiscovery starts peer discovery and returns channels for new and lost peers.
func StartPeerDiscovery(id string) (<-chan string, <-chan string) {
	peerUpdateCh := make(chan peers.PeerUpdate, 1)
	newPeerCh := make(chan string, 1)
	lostPeerCh := make(chan string, 1)

	peerBroadcastEnabledCh := make(chan bool)
	go peers.Transmitter(10001, id, peerBroadcastEnabledCh)
	go peers.Receiver(10001, peerUpdateCh)

	go func() {
		for update := range peerUpdateCh {
			fmt.Printf("Peer update: peers=%q new=%q lost=%q\n", update.Peers, update.New, update.Lost)
			if update.New != "" {
				newPeerCh <- update.New
			}
			for _, lostId := range update.Lost {
				lostPeerCh <- lostId
			}
		}
	}()

	return newPeerCh, lostPeerCh
}

// StartWorldviewBroadcast creates channels and starts the broadcast sender/receiver for worldview.
func StartWorldviewBroadcast() (chan<- wv.Worldview, <-chan wv.Worldview) {
	broadcastTx := make(chan wv.Worldview)
	broadcastRx := make(chan wv.Worldview)

	go bcast.Transmitter(10002, broadcastTx)
	go bcast.Receiver(10002, broadcastRx)

	return broadcastTx, broadcastRx
}
