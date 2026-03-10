package main

import (
	"Project/Network/setup"
	"Project/worldview"
	//"flag"
	"fmt"
	//"os"
	//"time"
)

func main() {

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

	go setup.TransmitWorldviewPeriodically(worldviewTx, worldviewToNetworkCh)

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



/*

Channels



updatedWorldviewToNetworkCh := make (chan Worldview)
updatedWorldviewToAssignerCh := make (chan Worldview)
updatedWorldviewToSyncCh := make (chan Worldview)

elevatorToWorldviewCh := make (chan StateEleator)
networkToWorldviewCh := make (chan Worldview)
syncToWorldviewCh := make (chan HallOrder)

newPeerIdCh := make (chan string) 
lostPeerIdCh := make (chan string)

worldviewToNetworkCh := make (chan ap[string]TransferWorldview)
worldviewToAssignerCh := make (chan map[int]Worldview)
worldviewToSyncCh := make (chan map[int]Worldview)


*/