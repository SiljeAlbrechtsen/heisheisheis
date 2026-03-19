package main

import (
	elevio "Project/Driver"
	fsm "Project/FSM"
	hardware "Project/Hardware"
	"Project/Network/setup"
	assign "Project/assignment"
	sync "Project/synchronization"
	wv "Project/worldview"
	"fmt" 
)

func main() {
	addr := setup.ResolveElevatorAddr()
	elevio.Init(addr, 4)

	// `go run main.go -id=our_id`
	id := setup.GetNodeID()

	// Channels into worldview
	elevatorStateCh := make(chan fsm.ElevatorState, 1)
	syncedHallOrdersCh := make(chan wv.HallOrders, 1)
	peerWorldviewCh := make(chan wv.Worldview, 1)
	initialWorldviewCh := make(chan wv.Worldview, 1)
	assignmentCh := make(chan map[string]wv.AssignmentMatrix, 1)
	cabRequestCh := make(chan int, 8)
	hallRequestCh := make(chan [2]int, 8)

	lightStateCh := make(chan wv.Worldview, 1)
	debugPrintReqCh := make(chan bool, 1)

	// Channels out of worldview
	worldviewsForAssignerCh := make(chan map[string]wv.Worldview, 1)
	worldviewsForSyncCh := make(chan map[string]wv.Worldview, 1)
	localWorldviewBroadcastCh := make(chan wv.Worldview, 1)
	fsmWorldviewCh := make(chan wv.Worldview, 16)
	newPeerCh, lostPeerCh := setup.StartPeerDiscovery(id)

	// Channels for network
	broadcastTx, broadcastRx := setup.SetupWorldviewNetwork()

	go hardware.ButtonLightsListener(lightStateCh)
	go hardware.ButtonsListener(cabRequestCh, hallRequestCh)
	go setup.TransmitWorldviewPeriodically(broadcastTx, localWorldviewBroadcastCh)
	go sync.GoroutineSync(id, syncedHallOrdersCh, worldviewsForSyncCh)
	go wv.RunWorldview(id, wv.WorldviewChannels{
		ElevatorState:  elevatorStateCh,
		SyncHallOrders: syncedHallOrdersCh,
		PeerWorldview:  peerWorldviewCh,
		InitWorldview:  initialWorldviewCh,
		LostPeer:       lostPeerCh,
		NewPeer:        newPeerCh,
		CabBtn:         cabRequestCh,
		HallBtn:        hallRequestCh,
		Assignment:     assignmentCh,
		PrintDebug:     debugPrintReqCh,
		Lights:         lightStateCh,
		ToAssigner:     worldviewsForAssignerCh,
		ToSync:         worldviewsForSyncCh,
		ToNetwork:      localWorldviewBroadcastCh,
		ToFSM:          fsmWorldviewCh,
	})
	go assign.RunHallRequestAssigner(id, worldviewsForAssignerCh, assignmentCh)
	go fsm.RunElevator(fsmWorldviewCh, elevatorStateCh, debugPrintReqCh)
	go setup.ForwardWorldviewFromNetwork(broadcastRx, peerWorldviewCh, initialWorldviewCh)

	fmt.Println("Started")
	select {}
}
