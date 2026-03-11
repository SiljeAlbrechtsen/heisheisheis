package main

import (
	fsm "Project/FSM"
	hardware "Project/Hardware"
	"Project/Network/setup"
	assign "Project/assignment"
	sync "Project/synchronization"
	wv "Project/worldview"

	//"flag"
	"fmt"
	//"os"
	//"time"
)

// TODO: må vi ha strl på input-variabler?

func main() {

	// `go run main.go -id=our_id`
	id := setup.GetNodeID()

	// CHANNELS
	// Må gjøre worldview private

	//To worldview
	elevatorToWorldviewCh := make(chan fsm.ElevatorState)
	syncToWorldviewCh := make(chan wv.HallOrders)
	networkToWorldviewCh := make(chan wv.Worldview)
	assignerToWordviewCh := make(chan map[string][4][3]bool, 1)
	cabBtnCh := make(chan int)
	hallBtnCh := make(chan [2]int)

	//From worldview
	worldviewToAssignerCh := make(chan map[string]wv.Worldview)
	worldviewToSyncCh := make(chan map[string]wv.Worldview)
	worldviewToNetworkCh := make(chan wv.Worldview)

	//From Sync
	lightOnCh := make(chan [2]int)
	lightsOffCh := make(chan [2]int)

	// From assigner
	assignerToFsmCh := make(chan [4][3]bool, 1) //Hardkodet ENDRE

	// Endret: peerUpdateCh returneres ikke lenger, se setup.go
	_, lostPeerIdCh := setup.StartPeerDiscovery(id)

	worldviewTx, worldviewRx := setup.SetupWorldviewNetwork()

	go hardware.ButtonsListener(cabBtnCh, hallBtnCh)

	go setup.TransmitWorldviewPeriodically(worldviewTx, worldviewToNetworkCh)

	go sync.GoRoutineSync(id, syncToWorldviewCh, worldviewToSyncCh, lightOnCh, lightsOffCh)

	go func() {
		for {
			select {
			case btn := <-lightOnCh:
				_ = btn
			case btn := <-lightsOffCh:
				_ = btn
			}
		}
	}()

	go wv.GoroutineForWorldview(id, elevatorToWorldviewCh, syncToWorldviewCh, networkToWorldviewCh, lostPeerIdCh, cabBtnCh, hallBtnCh, assignerToWordviewCh, worldviewToAssignerCh, worldviewToSyncCh, worldviewToNetworkCh)

	go assign.RunHallRequestAssigner(id, worldviewToAssignerCh, assignerToFsmCh, assignerToWordviewCh)

	go fsm.FSM2(assignerToFsmCh, elevatorToWorldviewCh)

	fmt.Println("Started")

	for {
		select {
		// Endret: peer update printing er flyttet til setup.go
		// case p := <-peerUpdateCh:
		// 	fmt.Printf("Peer update:\n")
		// 	fmt.Printf("  Peers:    %q\n", p.Peers)
		// 	fmt.Printf("  New:      %q\n", p.New)
		// 	fmt.Printf("  Lost:     %q\n", p.Lost)

		case a := <-worldviewRx:
			fmt.Printf("Received from %q: %#v\n", id, a)
			networkToWorldviewCh <- a
		}
	}

}

/*

CHANNELS:

updatedWorldviewToNetworkCh := make (chan Worldview)
updatedWorldviewToAssignerCh := make (chan Worldview)
updatedWorldviewToSyncCh := make (chan Worldview)

elevatorToWorldviewCh := make (chan StateEleator)
networkToWorldviewCh := make (chan Worldview)
syncToWorldviewCh := make (chan HallOrder)

newPeerIdCh := make (chan string)
lostPeerIdCh := makbuttonse (chan string)

worldviewToNetworkCh := make (chan ap[string]TransferWorldview)
worldviewToAssignerCh := make (chan map[int]Worldview)
worldviewToSyncCh := make (chan map[int]Worldview)


*/
