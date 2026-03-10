package main

import (
	"Project/Network/setup"
	//"flag"
	"fmt"
	//"os"
	//"time"
)

func main() {

	//__________________________________________________________________
	//----------------  SETTER UNIK ID FOR DENNE NODEN -----------------
	//__________________________________________________________________

	// `go run main.go -id=our_id`
	id := setup.GetNodeID()

	//__________________________________________________________________
	//---------------------- PEER DISCOVERY ----------------------------
	//__________________________________________________________________

	peerUpdateCh := setup.StartPeerDiscovery(id)

	//__________________________________________________________________
	//------------- STARTER KOMMUNIKASJON MED HEARTBEATS ---------------
	//__________________________________________________________________

	worldviewTx, worldviewRx := setup.SetupWorldviewNetwork()

	//__________________________________________________________________
	//----------- SENDER DENNE NODEN SINE HEARTBEATS PERIODISK ---------
	//__________________________________________________________________

	worldviewToNetworkCh := make(chan setup.Worldview)

	go setup.TransmittiWorldviewPeriodically(worldviewTx, worldviewToNetworkCh)


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

