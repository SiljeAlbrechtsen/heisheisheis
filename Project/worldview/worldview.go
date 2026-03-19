package worldview

import (
	fsm "Project/FSM"
	t "Project/types"
	"fmt"
	"time"
)

const (
	Directions = 2
	NumFloors  = 4
)

// Spesielle OwnerID-verdier brukt i synkroniseringsprotokollen
const (
	PeerDied = "peerDied"
	NoOwner  = ""
)

type OrderSyncState = t.OrderSyncState

const (
	None           = t.None
	Unconfirmed    = t.Unconfirmed
	Confirmed      = t.Confirmed
	DeleteProposed = t.DeleteProposed
)

type Order = t.Order

type HallOrders = t.HallOrders

// Worldview is now imported from types
type Worldview = t.Worldview

func worldviewInit(myId string, myWorldview Worldview, networkToInitCh <-chan Worldview) Worldview {
	myWv := myWorldview
	timeout := time.After(1 * time.Second)
	for {
		select {
		// Hvis den får andre worldvies
		case incomingWv := <-networkToInitCh:
			//Ignorerer seg selv
			if incomingWv.IdElevator == myId {
				continue
			}

			// Dyp kopi av cab orders (map-tilordning kopierer bare pekeren) Endret til deep copy
			myWv.AllCabOrders = make(map[string][NumFloors]bool, len(incomingWv.AllCabOrders))
			for id, orders := range incomingWv.AllCabOrders {
				myWv.AllCabOrders[id] = orders
			}
			myWv.HallOrders = incomingWv.HallOrders

			return myWv // ferdig init

		// Hvis de ikke får noe fra andre
		case <-timeout:
			return myWv
		}

	}

}

// _____________________________________________________________________________
// ----------FUNKSJONER FOR Å TA IMOT OG HÅNDTERE DATA FRA ANDRE MODULER--------
// _____________________________________________________________________________

// shouldAcceptSyncOrder avgjør om sync-resultatet er gyldig fremgang
// og ikke et stale resultat som ville regresse lokal tilstand.
func shouldAcceptSyncOrder(localOrder, syncOrder Order) bool {
	// Peer-death: ikke la stale Confirmed overskrive Unconfirmed fra peer-death,
	// men aksepter legitim konsensus (OwnerID="" betyr at sync faktisk avanserte)
	if localOrder.OwnerID == PeerDied && localOrder.SyncState == Unconfirmed &&
		syncOrder.SyncState == Confirmed && syncOrder.OwnerID != NoOwner {
		return false
	}

	// Samme state: alltid OK
	if syncOrder.SyncState == localOrder.SyncState {
		return true
	}

	// Fremover i syklusen: sync >= local (numerisk)
	if syncOrder.SyncState > localOrder.SyncState {
		return true
	}

	// Syklus-fullføring: DeleteProposed → None (konsensus)
	if localOrder.SyncState == DeleteProposed && syncOrder.SyncState == None {
		return true
	}

	// PeerDied-degradering: Confirmed → Unconfirmed/peerDied er tillatt
	// (eieren av ordren gikk i error, sync propagerer dette via Steg 0)
	if localOrder.SyncState == Confirmed &&
		syncOrder.SyncState == Unconfirmed &&
		syncOrder.OwnerID == PeerDied {
		return true
	}

	// Alt annet er stale — behold lokal tilstand
	return false
}

func updateWorldviewFromSync(latestWorldviews map[string]Worldview, inputSyncedHallOrders HallOrders, myID string) map[string]Worldview {
	worldviewsMap := latestWorldviews
	worldview := worldviewsMap[myID]

	for f := 0; f < NumFloors; f++ {
		for d := 0; d < Directions; d++ {
			localOrder := worldview.HallOrders[f][d]
			syncOrder := inputSyncedHallOrders[f][d]

			if !shouldAcceptSyncOrder(localOrder, syncOrder) {
				// Stale sync-resultat — behold lokal tilstand
				inputSyncedHallOrders[f][d] = localOrder
				continue
			}

			// Bevar lokalt satt OwnerID når sync ikke endrer SyncState (aldri for None-ordrer)
			if syncOrder.SyncState == localOrder.SyncState &&
				localOrder.OwnerID != NoOwner &&
				localOrder.SyncState != None {
				inputSyncedHallOrders[f][d].OwnerID = localOrder.OwnerID
			}
		}
	}

	worldview.HallOrders = inputSyncedHallOrders
	worldviewsMap[myID] = worldview
	return worldviewsMap
}

// Får inn worldview fra network, bruker IDen til å legge til/oppdatere map
func updatePeerWorldviewFromNetwork(latestWorldviews map[string]Worldview, inputPeerWorldview Worldview) map[string]Worldview {
	worldviewsMap := latestWorldviews
	peerID := inputPeerWorldview.IdElevator
	worldviewsMap[peerID] = inputPeerWorldview

	return worldviewsMap
}

// markPeerDeadInHallOrders degraderer Confirmed-ordrer eid av lostId til Unconfirmed/PeerDied,
// slik at andre heiser kan ta over ordren.
func markPeerDeadInHallOrders(hallOrders HallOrders, lostId string) HallOrders {
	ho := hallOrders
	for i, row := range ho {
		for j := range row {
			order := ho[i][j]
			if order.OwnerID == lostId && order.SyncState == Confirmed {
				order.SyncState = Unconfirmed
				order.OwnerID = PeerDied
			}
			ho[i][j] = order
		}
	}
	return ho
}

// updateWorldviewWithElevatorState oppdaterer worldview med ny elevatortilstand fra FSM,
// inkludert serverte cab-ordrer og fullførte hall-ordrer (setter DeleteProposed).
func updateWorldviewWithElevatorState(worldview Worldview, inputStateElevator t.ElevatorState, myID string) Worldview {
	wv := worldview
	prevState := wv.State
	wv.State = inputStateElevator
	floor := inputStateElevator.Floor

	if floor < 0 || floor >= NumFloors {
		return wv
	}

	if wv.AllCabOrders != nil {
		orders := wv.AllCabOrders[myID]
		if orders[floor] {
			orders[floor] = false
			wv.AllCabOrders[myID] = orders
		}
	}

	// Sjekk for servede hall-ordrer:
	// Case 1: FSM er DoorOpen nå (normal case - sjekk nåværende etasje)
	// Case 2: FSM VAR DoorOpen og har akkurat lukket døren (fanger opp missed clears)
	checkFloor := -1
	if inputStateElevator.Behaviour == fsm.EB_DoorOpen {
		checkFloor = floor
	} else if prevState.Behaviour == fsm.EB_DoorOpen {
		checkFloor = prevState.Floor
	}

	if checkFloor < 0 || checkFloor >= NumFloors {
		return wv
	}

	upOrder := wv.HallOrders[checkFloor][fsm.B_HallUp]
	if upOrder.SyncState == Confirmed &&
		!inputStateElevator.Requests[checkFloor][fsm.B_HallUp] &&
		(prevState.Requests[checkFloor][fsm.B_HallUp] || upOrder.OwnerID == myID) {
		upOrder.SyncState = DeleteProposed
		wv.HallOrders[checkFloor][fsm.B_HallUp] = upOrder
	}

	downOrder := wv.HallOrders[checkFloor][fsm.B_HallDown]
	if downOrder.SyncState == Confirmed &&
		!inputStateElevator.Requests[checkFloor][fsm.B_HallDown] &&
		(prevState.Requests[checkFloor][fsm.B_HallDown] || downOrder.OwnerID == myID) {
		downOrder.SyncState = DeleteProposed
		wv.HallOrders[checkFloor][fsm.B_HallDown] = downOrder
	}

	return wv
}

// updateOwnerIDsFromAssignment oppdaterer OwnerID på bekreftede hall-ordrer basert på assigner-resultatet.
func updateOwnerIDsFromAssignment(hallOrders HallOrders, assignment map[string][4][3]bool) HallOrders {
	ho := hallOrders
	for floor := 0; floor < NumFloors; floor++ {
		for dir := 0; dir < Directions; dir++ {
			if ho[floor][dir].SyncState == Confirmed &&
				(ho[floor][dir].OwnerID == NoOwner || ho[floor][dir].OwnerID == PeerDied) {
				for elevatorID, assigned := range assignment {
					if assigned[floor][dir] {
						ho[floor][dir].OwnerID = elevatorID
						break
					}
				}
			}
		}
	}

	return ho
}

func debugOrderSyncState(syncState OrderSyncState) string {
	switch syncState {
	case None:
		return "None"
	case Unconfirmed:
		return "Unconfirmed"
	case Confirmed:
		return "Confirmed"
	case DeleteProposed:
		return "DeleteProposed"
	default:
		return fmt.Sprintf("Unknown(%d)", syncState)
	}
}

func debugHallDirection(dir int) string {
	switch dir {
	case 0:
		return "Up"
	case 1:
		return "Down"
	default:
		return fmt.Sprintf("Unknown(%d)", dir)
	}
}

func debugPrintHallOrders(context string, hallOrders HallOrders) {
	fmt.Printf("\n[Worldview] Hallorders %s\n", context)
	for floor := NumFloors - 1; floor >= 0; floor-- {
		fmt.Printf("  Floor %d:\n", floor)
		for dir := 0; dir < Directions; dir++ {
			order := hallOrders[floor][dir]
			fmt.Printf("    %-4s state=%-14s owner=%q\n", debugHallDirection(dir), debugOrderSyncState(order.SyncState), order.OwnerID)
		}
	}
}

func GoroutineForWorldview(
	myID string,
	elevatorToWorldviewCh <-chan t.ElevatorState,
	syncToWorldviewCh <-chan HallOrders,
	networkToWorldviewCh <-chan Worldview,
	networkToInitCh <-chan Worldview,

	lostPeerIdCh <-chan string,
	newPeerIdCh <-chan string,
	cabBtnCh <-chan int,
	hallBtnCh <-chan [2]int,
	lightsCh chan Worldview,
	printHallOrdersReqCh <-chan bool,

	assignerToWorldviewCh <-chan map[string][4][3]bool,
	worldviewToAssignerCh chan map[string]Worldview,
	worldviewToSyncCh chan map[string]Worldview,
	worldviewToNetworkCh chan Worldview,
	worldviewToFSMCh chan Worldview,
) {

	worldviewsMap := make(map[string]Worldview)
	initialWv := Worldview{
		IdElevator:   myID,
		AllCabOrders: map[string][NumFloors]bool{myID: {}},
	}
	worldviewsMap[myID] = worldviewInit(myID, initialWv, networkToInitCh)

	hasNetwork := true

	copyMap := func(m map[string]Worldview) map[string]Worldview {
		c := make(map[string]Worldview, len(m))
		for k, v := range m {
			if v.AllCabOrders != nil {
				newAllCabOrders := make(map[string][NumFloors]bool, len(v.AllCabOrders))
				for id, orders := range v.AllCabOrders {
					newAllCabOrders[id] = orders
				}
				v.AllCabOrders = newAllCabOrders
			}
			c[k] = v
		}
		return c
	}

	sendLatestLights := func() {
		wv := copyMap(worldviewsMap)[myID]
		select {
		case lightsCh <- wv:
		default:
			select {
			case <-lightsCh:
			default:
			}
			lightsCh <- wv
		}
	}

	sendLatestWorldviewToFSM := func(worldview Worldview) {
		select {
		case worldviewToFSMCh <- worldview:
		default:
			select {
			case <-worldviewToFSMCh:
			default:
			}
			worldviewToFSMCh <- worldview
		}
	}

	sendLatestToNetwork := func(worldview Worldview) {
		select {
		case worldviewToNetworkCh <- worldview:
		default:
			select {
			case <-worldviewToNetworkCh:
			default:
			}
			worldviewToNetworkCh <- worldview
		}
	}

	sendLatestToSync := func(worldviews map[string]Worldview) {
		select {
		case worldviewToSyncCh <- worldviews:
		default:
			select {
			case <-worldviewToSyncCh:
			default:
			}
			worldviewToSyncCh <- worldviews
		}
	}

	sendLatestToAssigner := func(worldviews map[string]Worldview) {
		select {
		case worldviewToAssignerCh <- worldviews:
		default:
			select {
			case <-worldviewToAssignerCh:
			default:
			}
			worldviewToAssignerCh <- worldviews
		}
	}

	for {
		select {
		case inputStateElevator := <-elevatorToWorldviewCh:
			wv := worldviewsMap[myID]
			wv = updateWorldviewWithElevatorState(wv, inputStateElevator, myID)
			if wv.AllCabOrders == nil {
				wv.AllCabOrders = make(map[string][NumFloors]bool)
			}
			if inputStateElevator.Error {
				wv.ErrorState = true
				wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, myID)
			} else {
				wv.ErrorState = false
			}
			worldviewsMap[myID] = wv
			sendLatestLights()
			sendLatestToNetwork(copyMap(worldviewsMap)[myID])
			sendLatestToSync(copyMap(worldviewsMap))

		case inputSyncedHallOrders := <-syncToWorldviewCh:
			worldviewsMap = updateWorldviewFromSync(worldviewsMap, inputSyncedHallOrders, myID)
			sendLatestToAssigner(copyMap(worldviewsMap))
			sendLatestLights()
			sendLatestToNetwork(copyMap(worldviewsMap)[myID])

		case inputPeerWorldview := <-networkToWorldviewCh:
			if inputPeerWorldview.IdElevator == myID {
				continue
			}
			worldviewsMap = updatePeerWorldviewFromNetwork(worldviewsMap, inputPeerWorldview)
			wv := worldviewsMap[myID]
			if wv.AllCabOrders == nil {
				wv.AllCabOrders = make(map[string][NumFloors]bool)
			}
			wv.AllCabOrders[inputPeerWorldview.IdElevator] = inputPeerWorldview.AllCabOrders[inputPeerWorldview.IdElevator]
			if inputPeerWorldview.ErrorState {
				wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, inputPeerWorldview.IdElevator)
			}
			worldviewsMap[myID] = wv
			sendLatestToNetwork(copyMap(worldviewsMap)[myID])
			sendLatestToSync(copyMap(worldviewsMap))

		case newPeerID := <-newPeerIdCh:
			fmt.Printf("[Worldview] Ny peer oppdaget: %s\n", newPeerID)
			if newPeerID == myID {
				hasNetwork = true
				// Gjenopprett hallOrders fra en kjent peer ved reconnect
				wv := worldviewsMap[myID]
				for id, peerWv := range worldviewsMap {
					if id != myID {
						wv.HallOrders = peerWv.HallOrders
						break
					}
				}
				worldviewsMap[myID] = wv
			}

		case inputDeadPeerId := <-lostPeerIdCh:
			if inputDeadPeerId == myID {
				hasNetwork = false
			}
			worldviewsMap = handleLostPeer(worldviewsMap, myID, inputDeadPeerId)
			sendLatestToNetwork(copyMap(worldviewsMap)[myID])
			sendLatestToSync(copyMap(worldviewsMap))

		case inputHallBtn := <-hallBtnCh:
			if hasNetwork {
				worldviewsMap[myID] = addNewHallOrder(worldviewsMap[myID], inputHallBtn)
				sendLatestLights()
				sendLatestToNetwork(copyMap(worldviewsMap)[myID])
				sendLatestToSync(copyMap(worldviewsMap))
			}

		case inputCabBtn := <-cabBtnCh:
			worldviewsMap[myID] = addNewCabOrder(worldviewsMap[myID], inputCabBtn, myID)
			sendLatestLights()
			sendLatestToNetwork(copyMap(worldviewsMap)[myID])
			sendLatestToSync(copyMap(worldviewsMap))

		case inputAssignment := <-assignerToWorldviewCh:
			wv := worldviewsMap[myID]
			wv.HallOrders = updateOwnerIDsFromAssignment(wv.HallOrders, inputAssignment)
			worldviewsMap[myID] = wv
			sendLatestLights()
			sendLatestToNetwork(copyMap(worldviewsMap)[myID])
			sendLatestWorldviewToFSM(copyMap(worldviewsMap)[myID])

		case <-printHallOrdersReqCh:
			debugPrintHallOrders("stop button worldview", worldviewsMap[myID].HallOrders)
		}
	}
}

// handleLostPeer markerer tapt peer som død og degraderer dens ordrer til Unconfirmed/PeerDied.
func handleLostPeer(latestWorldviews map[string]Worldview, myID string, lostID string) map[string]Worldview {
	if lostID == myID {
		return latestWorldviews
	}
	lwv := latestWorldviews
	lostWorldview := lwv[lostID]
	lostWorldview.Dead = true
	lwv[lostID] = lostWorldview

	wv := lwv[myID]
	wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, lostID)

	lwv[myID] = wv

	return lwv
}

func addNewCabOrder(worldview Worldview, inputCabBtn int, myID string) Worldview {
	if inputCabBtn < 0 || inputCabBtn >= NumFloors {
		return worldview
	}
	wv := worldview

	cabOrders := wv.AllCabOrders[myID]
	cabOrders[inputCabBtn] = true
	wv.AllCabOrders[myID] = cabOrders

	return wv
}

func addNewHallOrder(worldview Worldview, inputHallBtn [2]int) Worldview {
	floor := inputHallBtn[0]
	dir := inputHallBtn[1]
	if floor < 0 || floor >= NumFloors || dir < 0 || dir >= Directions {
		return worldview
	}
	wv := worldview

	order := wv.HallOrders[floor][dir]

	if order.SyncState == None {
		order.SyncState = Unconfirmed
		order.OwnerID = NoOwner
	}
	wv.HallOrders[floor][dir] = order

	return wv
}
