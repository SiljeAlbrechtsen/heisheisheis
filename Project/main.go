package main

import (
	fsm "Project/FSM"
	hardware "Project/Hardware"
	"Project/Network/setup"
	assign "Project/assignment"
	sync "Project/synchronization"
	wv "Project/worldview"
	"time"

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
	syncToWorldviewCh := make(chan wv.HallOrders, 1)
	networkToWorldviewCh := make(chan wv.Worldview)
	assignerToWordviewCh := make(chan map[string][4][3]bool, 1)
	cabBtnCh := make(chan int, 8)
	hallBtnCh := make(chan [2]int, 8)

	//From worldview
	worldviewToAssignerCh := make(chan map[string]wv.Worldview, 1)
	worldviewToSyncCh := make(chan map[string]wv.Worldview, 1)
	worldviewToNetworkCh := make(chan wv.Worldview, 1)

	//From Sync
	lightOnCh := make(chan [2]int, 16)
	lightsOffCh := make(chan [2]int, 16)

	// From assigner
	assignerToFsmCh := make(chan [4][3]bool, 16) //Hardkodet ENDRE Litt ekstremt med 16 i buffer?

	// Endret: peerUpdateCh returneres ikke lenger, se setup.go
	_, lostPeerIdCh := setup.StartPeerDiscovery(id)

	worldviewTx, worldviewRx := setup.SetupWorldviewNetwork()

	_ = fsm.InitElevatorState()

	go hardware.ButtonsListener(cabBtnCh, hallBtnCh) // Good

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

	go func() {
		for {
			<-assignerToWordviewCh // kanal som aldri blir brukt
		}
	}()

	go fsm.FSM2(assignerToFsmCh, elevatorToWorldviewCh)

	fmt.Println("Started")

	for {
		select {
		case a := <-worldviewRx:
			//fmt.Printf("Received from %q: %#v\n", id, a)
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
func main9() {
	elevatorToWorldviewCh := make(chan fsm.ElevatorState)
	assignerToFsmCh := make(chan [4][3]bool) //Hardkodet ENDRE

	//assign.RunHallRequestAssigner(id, worldviewToAssignerCh, assignerToFsmCh, assignerToWordviewCh)
	fakeMap := [4][3]bool{
		{false, false, false},
		{false, false, false},
		{false, true, false},
		{false, false, false},
	}

	go fsm.FSM2(assignerToFsmCh, elevatorToWorldviewCh)

	go func() {
		for state := range elevatorToWorldviewCh {
			fmt.Printf("Elevator state update: floor=%d dir=%d behaviour=%d\n", state.Floor, state.Dirn, state.Behaviour)
		}
	}()

	for {
		time.Sleep(5 * time.Second)
		assignerToFsmCh <- fakeMap
	}
}


/*
Suggested next hardening

Keep channels buffered for pipeline stages.
Optionally use non-blocking send for telemetry-like updates (network/sync snapshots) if freshness is more important than delivery of every intermediate state.
Remove temporary debug prints in worldview/FSM once stable so logs show real stalls clearly.
If you want, I can do one more pass to make worldview fan-out explicitly non-blocking (drop stale updates safely) so this cannot lock under load.
*/