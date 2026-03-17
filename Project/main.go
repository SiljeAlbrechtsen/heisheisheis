package main

//A-To do: Fikse at lights on sender caborders lys, sender kun hallorders atm. Også sender off signal dårlig.

import (
	fsm "Project/FSM"
	hardware "Project/Hardware"
	"Project/Network/setup"
	assign "Project/assignment"
	sync "Project/synchronization"
	t "Project/types"
	wv "Project/worldview"

	//"flag"
	"fmt"
	//"os"
	//"time"
)

// TODO: må vi ha strl på input-variabler?

func main() {
	t.InitDriver()

	// `go run main.go -id=our_id`
	id := setup.GetNodeID()

	// CHANNELS
	// Må gjøre worldview private

	//To worldview
	elevatorToWorldviewCh := make(chan fsm.ElevatorState, 1) //A-La til buffer, idk why
	syncToWorldviewCh := make(chan wv.HallOrders, 1)
	networkToWorldviewCh := make(chan wv.Worldview, 1)
	networkToInitCh := make(chan wv.Worldview, 1)
	assignerToWordviewCh := make(chan map[string][4][3]bool, 1)
	cabBtnCh := make(chan int, 8)
	hallBtnCh := make(chan [2]int, 8)

	//From worldview
	worldviewToAssignerCh := make(chan map[string]wv.Worldview, 1)
	worldviewToSyncCh := make(chan map[string]wv.Worldview, 1)
	worldviewToNetworkCh := make(chan wv.Worldview, 1)

	//From Sync
	lightOnCh := make(chan [2]int)
	lightsOffCh := make(chan [2]int)

	// From assigner
	assignerToFsmCh := make(chan [4][3]bool, 1)

	// Endret: peerUpdateCh returneres ikke lenger, se setup.go
	newPeerIdCh, lostPeerIdCh := setup.StartPeerDiscovery(id)

	worldviewTx, worldviewRx := setup.SetupWorldviewNetwork()

	go hardware.ButtonsListener(cabBtnCh, hallBtnCh)

	go setup.TransmitWorldviewPeriodically(worldviewTx, worldviewToNetworkCh)

	go sync.GoRoutineSync(id, syncToWorldviewCh, worldviewToSyncCh, lightOnCh, lightsOffCh)

	go hardware.LightsListener(lightOnCh, lightsOffCh)

	go wv.GoroutineForWorldview(id, elevatorToWorldviewCh, syncToWorldviewCh, networkToWorldviewCh, networkToInitCh, lostPeerIdCh, newPeerIdCh, cabBtnCh, hallBtnCh, assignerToWordviewCh, worldviewToAssignerCh, worldviewToSyncCh, worldviewToNetworkCh)

	go assign.RunHallRequestAssigner(id, worldviewToAssignerCh, assignerToFsmCh, assignerToWordviewCh)

	go fsm.FSM3(assignerToFsmCh, elevatorToWorldviewCh)

	go setup.ForwardWorldviewFromNetwork(worldviewRx, networkToWorldviewCh, networkToInitCh)

	fmt.Println("Started")

	select{}
/*
	for a := range worldviewRx {
		networkToInitCh <- a
		networkToWorldviewCh <- a
	}
*/
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
