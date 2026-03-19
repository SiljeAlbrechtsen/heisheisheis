package main

import (
	elevio "Project/Driver"
	fsm "Project/FSM"
	hardware "Project/Hardware"
	"Project/Network/setup"
	assign "Project/assignment"
	sync "Project/synchronization"
	elev "Project/elevator"
	wv "Project/worldview"

	"fmt"
)

// Architecture: peer-to-peer distributed elevator control.
//
// Worldview is the state hub — all modules communicate through it.
// A hall order follows this path before the FSM serves it:
//
//   Button → Worldview → Sync (4-stage consensus) → Worldview
//                      → Assigner (cost-based assignment) → Worldview → FSM
//
// Sync cycle: None → Unconfirmed → Confirmed → DeleteProposed → None.
// FSM only serves orders where state is Confirmed and OwnerID == own ID.
func main() {
	addr := setup.ResolveElevatorAddr()
	elevio.Init(addr, 4)

	// `go run main.go -id=our_id`
	id := setup.GetNodeID()

	// Channels inn til worldview
	elevatorToWorldviewCh := make(chan fsm.ElevatorState, 1)
	syncToWorldviewCh := make(chan wv.HallOrders, 1)
	networkToWorldviewCh := make(chan wv.Worldview, 1)
	networkToInitCh := make(chan wv.Worldview, 1)
	assignerToWorldviewCh := make(chan map[string]wv.AssignmentMatrix, 1)
	cabBtnCh := make(chan int, 8)
	hallBtnCh := make(chan [2]int, 8)

	lightsCh := make(chan wv.Worldview, 1)
	printHallOrdersReqCh := make(chan bool, 1)

	// Channels ut fra worldview
	worldviewToAssignerCh := make(chan map[string]wv.Worldview, 1)
	worldviewToSyncCh := make(chan map[string]wv.Worldview, 1)
	worldviewToNetworkCh := make(chan wv.Worldview, 1)
	worldviewToFSMCh := make(chan [elev.N_FLOORS][elev.N_BUTTONS]bool, 16)

	newPeerIdCh, lostPeerIdCh := setup.StartPeerDiscovery(id)

	worldviewTx, worldviewRx := setup.SetupWorldviewNetwork()

	go hardware.ButtonLightsListener(lightsCh)
	go hardware.ButtonsListener(cabBtnCh, hallBtnCh)
	go setup.TransmitWorldviewPeriodically(worldviewTx, worldviewToNetworkCh)
	go sync.GoroutineSync(id, syncToWorldviewCh, worldviewToSyncCh)
	go wv.GoroutineForWorldview(id, wv.WorldviewChannels{
		ElevatorState:  elevatorToWorldviewCh,
		SyncHallOrders: syncToWorldviewCh,
		PeerWorldview:  networkToWorldviewCh,
		InitWorldview:  networkToInitCh,
		LostPeer:       lostPeerIdCh,
		NewPeer:        newPeerIdCh,
		CabBtn:         cabBtnCh,
		HallBtn:        hallBtnCh,
		Assignment:     assignerToWorldviewCh,
		PrintDebug:     printHallOrdersReqCh,
		Lights:         lightsCh,
		ToAssigner:     worldviewToAssignerCh,
		ToSync:         worldviewToSyncCh,
		ToNetwork:      worldviewToNetworkCh,
		ToFSM:          worldviewToFSMCh,
	})
	go assign.RunHallRequestAssigner(id, worldviewToAssignerCh, assignerToWorldviewCh)
	go fsm.RunElevator(worldviewToFSMCh, elevatorToWorldviewCh, printHallOrdersReqCh)
	go setup.ForwardWorldviewFromNetwork(worldviewRx, networkToWorldviewCh, networkToInitCh)

	fmt.Println("Started")
	select {}
}
