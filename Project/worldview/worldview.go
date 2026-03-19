package worldview

import (
	fsm "Project/FSM"
	t "Project/types"
	"fmt"
	"time"
)

const (
	Directions = t.N_DIRECTIONS
	NumFloors  = t.N_FLOORS
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
type Worldview = t.Worldview

// WorldviewChannels grupperer alle kanaler inn og ut av worldview-goroutinen.
type WorldviewChannels struct {
	// Worldview leser fra disse
	ElevatorState  <-chan t.ElevatorState
	SyncHallOrders <-chan HallOrders
	PeerWorldview  <-chan Worldview
	InitWorldview  <-chan Worldview
	LostPeer       <-chan string
	NewPeer        <-chan string
	CabBtn         <-chan int
	HallBtn        <-chan [2]int
	Assignment     <-chan map[string][4][3]bool
	PrintDebug     <-chan bool

	// Worldview skriver til disse
	Lights     chan Worldview
	ToAssigner chan map[string]Worldview
	ToSync     chan map[string]Worldview
	ToNetwork  chan Worldview
	ToFSM      chan Worldview
}

// copyWorldviews lager en dyp kopi av worldviews-mapet (inkl. AllCabOrders).
func copyWorldviews(m map[string]Worldview) map[string]Worldview {
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

func worldviewInit(myID string, myWorldview Worldview, initCh <-chan Worldview) Worldview {
	myWv := myWorldview
	timeout := time.After(1 * time.Second)
	for {
		select {
		case incomingWv := <-initCh:
			if incomingWv.IdElevator == myID {
				continue
			}
			copied := copyWorldviews(map[string]Worldview{incomingWv.IdElevator: incomingWv})[incomingWv.IdElevator]
			myWv.HallOrders = copied.HallOrders
			myWv.AllCabOrders = copied.AllCabOrders
			return myWv

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

func updateWorldviewFromSync(worldviews map[string]Worldview, incomingOrders HallOrders, myID string) map[string]Worldview {
	wv := worldviews[myID]
	merged := incomingOrders // start fra incoming, overstyr avviste entries med lokal tilstand

	for f := 0; f < NumFloors; f++ {
		for d := 0; d < Directions; d++ {
			localOrder := wv.HallOrders[f][d]
			syncOrder := incomingOrders[f][d]

			if !shouldAcceptSyncOrder(localOrder, syncOrder) {
				// Stale sync-resultat — behold lokal tilstand
				merged[f][d] = localOrder
				continue
			}

			// Bevar lokalt satt OwnerID kun hvis sync ikke har en konkret eier.
			// Hvis sync har satt en konkret eier (f.eks. via konfliktløsning), bruk den.
			if syncOrder.SyncState == localOrder.SyncState &&
				localOrder.OwnerID != NoOwner &&
				localOrder.SyncState != None &&
				(syncOrder.OwnerID == NoOwner || syncOrder.OwnerID == PeerDied) {
				merged[f][d].OwnerID = localOrder.OwnerID
			}
		}
	}

	wv.HallOrders = merged
	worldviews[myID] = wv
	return worldviews
}

// applyPeerWorldview lagrer peerens worldview og oppdaterer egen tilstand basert på den:
// synkroniserer cab orders og degraderer hall orders hvis peeren er i error.
func applyPeerWorldview(worldviews map[string]Worldview, peerWv Worldview, myID string) map[string]Worldview {
	worldviews[peerWv.IdElevator] = peerWv

	wv := worldviews[myID]
	if wv.AllCabOrders == nil {
		wv.AllCabOrders = make(map[string][NumFloors]bool)
	}
	wv.AllCabOrders[peerWv.IdElevator] = peerWv.AllCabOrders[peerWv.IdElevator]
	if peerWv.ErrorState {
		wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, peerWv.IdElevator)
	}
	worldviews[myID] = wv
	return worldviews
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
// inkludert error-state, serverte cab-ordrer og fullførte hall-ordrer (setter DeleteProposed).
func updateWorldviewWithElevatorState(worldview Worldview, newState t.ElevatorState, myID string) Worldview {
	wv := worldview
	prevState := wv.State
	wv.State = newState
	wv.ErrorState = newState.Error
	if newState.Error {
		wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, myID)
	}
	floor := newState.Floor

	if wv.AllCabOrders == nil {
		wv.AllCabOrders = make(map[string][NumFloors]bool)
	}

	if floor < 0 || floor >= NumFloors {
		return wv
	}

	orders := wv.AllCabOrders[myID]
	if orders[floor] {
		orders[floor] = false
		wv.AllCabOrders[myID] = orders
	}

	// Sjekk for servede hall-ordrer:
	// Case 1: FSM er DoorOpen nå (normal case - sjekk nåværende etasje)
	// Case 2: FSM VAR DoorOpen og har akkurat lukket døren (fanger opp missed clears)
	checkFloor := -1
	if newState.Behaviour == fsm.EB_DoorOpen {
		checkFloor = floor
	} else if prevState.Behaviour == fsm.EB_DoorOpen {
		checkFloor = prevState.Floor
	}

	if checkFloor < 0 || checkFloor >= NumFloors {
		return wv
	}

	upOrder := wv.HallOrders[checkFloor][fsm.B_HallUp]
	if upOrder.SyncState == Confirmed &&
		!newState.Requests[checkFloor][fsm.B_HallUp] &&
		(prevState.Requests[checkFloor][fsm.B_HallUp] || upOrder.OwnerID == myID) {
		upOrder.SyncState = DeleteProposed
		wv.HallOrders[checkFloor][fsm.B_HallUp] = upOrder
	}

	downOrder := wv.HallOrders[checkFloor][fsm.B_HallDown]
	if downOrder.SyncState == Confirmed &&
		!newState.Requests[checkFloor][fsm.B_HallDown] &&
		(prevState.Requests[checkFloor][fsm.B_HallDown] || downOrder.OwnerID == myID) {
		downOrder.SyncState = DeleteProposed
		wv.HallOrders[checkFloor][fsm.B_HallDown] = downOrder
	}

	return wv
}

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

func GoroutineForWorldview(myID string, ch WorldviewChannels) {
	worldviews := make(map[string]Worldview)
	initialWv := Worldview{
		IdElevator:   myID,
		AllCabOrders: map[string][NumFloors]bool{myID: {}},
	}
	worldviews[myID] = worldviewInit(myID, initialWv, ch.InitWorldview)

	hasNetwork := true

	sendLights := func() {
		wv := copyWorldviews(worldviews)[myID]
		select {
		case ch.Lights <- wv:
		default:
			select {
			case <-ch.Lights:
			default:
			}
			ch.Lights <- wv
		}
	}

	sendToFSM := func(wv Worldview) {
		select {
		case ch.ToFSM <- wv:
		default:
			select {
			case <-ch.ToFSM:
			default:
			}
			ch.ToFSM <- wv
		}
	}

	sendToNetwork := func(wv Worldview) {
		select {
		case ch.ToNetwork <- wv:
		default:
			select {
			case <-ch.ToNetwork:
			default:
			}
			ch.ToNetwork <- wv
		}
	}

	sendToSync := func(wvs map[string]Worldview) {
		select {
		case ch.ToSync <- wvs:
		default:
			select {
			case <-ch.ToSync:
			default:
			}
			ch.ToSync <- wvs
		}
	}

	sendToAssigner := func(wvs map[string]Worldview) {
		select {
		case ch.ToAssigner <- wvs:
		default:
			select {
			case <-ch.ToAssigner:
			default:
			}
			ch.ToAssigner <- wvs
		}
	}

	for {
		select {
		case newState := <-ch.ElevatorState:
			worldviews[myID] = updateWorldviewWithElevatorState(worldviews[myID], newState, myID)
			sendLights()
			sendToNetwork(copyWorldviews(worldviews)[myID])
			sendToSync(copyWorldviews(worldviews))

		case syncedHallOrders := <-ch.SyncHallOrders:
			worldviews = updateWorldviewFromSync(worldviews, syncedHallOrders, myID)
			sendToAssigner(copyWorldviews(worldviews))
			sendLights()
			sendToNetwork(copyWorldviews(worldviews)[myID])

		case peerWv := <-ch.PeerWorldview:
			if peerWv.IdElevator == myID {
				continue
			}
			worldviews = applyPeerWorldview(worldviews, peerWv, myID)
			sendToNetwork(copyWorldviews(worldviews)[myID])
			sendToSync(copyWorldviews(worldviews))

		case newPeerID := <-ch.NewPeer:
			fmt.Printf("[Worldview] Ny peer oppdaget: %s\n", newPeerID)
			if newPeerID == myID {
				hasNetwork = true
				// Gjenopprett hallOrders fra en kjent peer ved reconnect
				wv := worldviews[myID]
				for id, peerWv := range worldviews {
					if id != myID {
						wv.HallOrders = peerWv.HallOrders
						break
					}
				}
				worldviews[myID] = wv
			}

		case lostPeerID := <-ch.LostPeer:
			if lostPeerID == myID {
				hasNetwork = false
			}
			worldviews = handleLostPeer(worldviews, myID, lostPeerID)
			sendToNetwork(copyWorldviews(worldviews)[myID])
			sendToSync(copyWorldviews(worldviews))

		case hallBtn := <-ch.HallBtn:
			if hasNetwork {
				worldviews[myID] = addNewHallOrder(worldviews[myID], hallBtn)
				sendLights()
				sendToNetwork(copyWorldviews(worldviews)[myID])
				sendToSync(copyWorldviews(worldviews))
			}

		case cabBtn := <-ch.CabBtn:
			worldviews[myID] = addNewCabOrder(worldviews[myID], cabBtn, myID)
			sendLights()
			sendToNetwork(copyWorldviews(worldviews)[myID])
			sendToSync(copyWorldviews(worldviews))

		case assignment := <-ch.Assignment:
			wv := worldviews[myID]
			wv.HallOrders = updateOwnerIDsFromAssignment(wv.HallOrders, assignment)
			worldviews[myID] = wv
			sendLights()
			sendToNetwork(copyWorldviews(worldviews)[myID])
			sendToFSM(copyWorldviews(worldviews)[myID])

		case <-ch.PrintDebug:
			debugPrintHallOrders("stop button worldview", worldviews[myID].HallOrders)
		}
	}
}

// handleLostPeer markerer tapt peer som død og degraderer dens ordrer til Unconfirmed/PeerDied.
func handleLostPeer(worldviews map[string]Worldview, myID string, lostID string) map[string]Worldview {
	if lostID == myID {
		return worldviews
	}
	lostWv := worldviews[lostID]
	lostWv.Dead = true
	worldviews[lostID] = lostWv

	wv := worldviews[myID]
	wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, lostID)
	worldviews[myID] = wv

	return worldviews
}

func addNewCabOrder(worldview Worldview, floor int, myID string) Worldview {
	if floor < 0 || floor >= NumFloors {
		return worldview
	}
	wv := worldview
	cabOrders := wv.AllCabOrders[myID]
	cabOrders[floor] = true
	wv.AllCabOrders[myID] = cabOrders
	return wv
}

func addNewHallOrder(worldview Worldview, btn [2]int) Worldview {
	floor := btn[0]
	dir := btn[1]
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
