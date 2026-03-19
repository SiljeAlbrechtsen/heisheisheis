package worldview

import (
	elev "Project/elevator"
	"fmt"
	"time"
)

type OrderSyncState int

const (
	None           OrderSyncState = iota
	Unconfirmed
	Confirmed
	DeleteProposed
)

// Special OwnerID sentinel values used in the synchronization protocol
const (
	PeerDied = "peerDied"
	NoOwner  = ""
)

type Order struct {
	SyncState OrderSyncState
	OwnerID   string
}

type HallOrders [elev.N_FLOORS][2]Order
type AssignmentMatrix [elev.N_FLOORS][elev.N_BUTTONS]bool

type Worldview struct {
	IdElevator   string
	HallOrders   HallOrders
	State        elev.ElevatorState
	AllCabOrders map[string][elev.N_FLOORS]bool
	ErrorState   bool
	Dead         bool
}

// WorldviewChannels groups all channels into and out of the worldview goroutine.
type WorldviewChannels struct {
	// Worldview reads from these
	ElevatorState  <-chan elev.ElevatorState
	SyncHallOrders <-chan HallOrders
	PeerWorldview  <-chan Worldview
	InitWorldview  <-chan Worldview
	LostPeer       <-chan string
	NewPeer        <-chan string
	CabBtn         <-chan int
	HallBtn        <-chan [2]int
	Assignment     <-chan map[string]AssignmentMatrix
	PrintDebug     <-chan bool

	// Worldview writes to these
	Lights     chan Worldview
	ToAssigner chan map[string]Worldview
	ToSync     chan map[string]Worldview
	ToNetwork  chan Worldview
	ToFSM      chan [elev.N_FLOORS][elev.N_BUTTONS]bool
}

func sendLatestWorldview(ch chan Worldview, v Worldview) {
	select {
	case ch <- v:
	default:
		select {
		case <-ch:
		default:
		}
		ch <- v
	}
}

func sendLatestWorldviewMap(ch chan map[string]Worldview, v map[string]Worldview) {
	select {
	case ch <- v:
	default:
		select {
		case <-ch:
		default:
		}
		ch <- v
	}
}

// copyWorldviews returns a deep copy of the worldviews map (including AllCabOrders).
func copyWorldviews(m map[string]Worldview) map[string]Worldview {
	c := make(map[string]Worldview, len(m))
	for k, v := range m {
		if v.AllCabOrders != nil {
			newAllCabOrders := make(map[string][elev.N_FLOORS]bool, len(v.AllCabOrders))
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

// shouldAcceptSyncOrder determines whether the sync result represents valid progress.
// Cycle: None(0) → Unconfirmed(1) → Confirmed(2) → DeleteProposed(3) → None(0)
func shouldAcceptSyncOrder(localOrder, syncOrder Order) bool {
	if syncOrder.SyncState == localOrder.SyncState {
		return true
	}

	// Forward in the cycle (numeric)
	// Exception: do not re-confirm an order we have already marked as PeerDied
	if syncOrder.SyncState > localOrder.SyncState {
		staleConfirm := localOrder.SyncState == Unconfirmed &&
			localOrder.OwnerID == PeerDied &&
			syncOrder.SyncState == Confirmed &&
			syncOrder.OwnerID != NoOwner
		return !staleConfirm
	}

	// Cycle completion: DeleteProposed → None
	if localOrder.SyncState == DeleteProposed && syncOrder.SyncState == None {
		return true
	}

	// PeerDied degradation: Confirmed → Unconfirmed/PeerDied
	if localOrder.SyncState == Confirmed &&
		syncOrder.SyncState == Unconfirmed &&
		syncOrder.OwnerID == PeerDied {
		return true
	}

	return false
}

func updateWorldviewFromSync(worldviews map[string]Worldview, incomingOrders HallOrders, myID string) map[string]Worldview {
	wv := worldviews[myID]
	merged := incomingOrders // start from incoming, override rejected entries with local state

	for f := 0; f < elev.N_FLOORS; f++ {
		for d := 0; d < elev.N_DIRECTIONS; d++ {
			localOrder := wv.HallOrders[f][d]
			syncOrder := incomingOrders[f][d]

			if !shouldAcceptSyncOrder(localOrder, syncOrder) {
				// Stale sync result — keep local state
				merged[f][d] = localOrder
				continue
			}

			// Preserve locally set OwnerID only if sync has no concrete owner.
			// If sync has a concrete owner (e.g. from conflict resolution), use it.
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

// applyPeerWorldview stores the peer's worldview and updates own state based on it:
// syncs cab orders and degrades hall orders if the peer is in error state.
func applyPeerWorldview(worldviews map[string]Worldview, peerWv Worldview, myID string) map[string]Worldview {
	worldviews[peerWv.IdElevator] = peerWv

	wv := worldviews[myID]
	if wv.AllCabOrders == nil {
		wv.AllCabOrders = make(map[string][elev.N_FLOORS]bool)
	}
	wv.AllCabOrders[peerWv.IdElevator] = peerWv.AllCabOrders[peerWv.IdElevator]
	if peerWv.ErrorState {
		wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, peerWv.IdElevator)
	}
	worldviews[myID] = wv
	return worldviews
}

// markPeerDeadInHallOrders degrades Confirmed orders owned by lostId to Unconfirmed/PeerDied,
// so that other elevators can take over the order.
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
func updateWorldviewWithElevatorState(worldview Worldview, newState elev.ElevatorState, myID string) Worldview {
	wv := worldview
	prevState := wv.State
	wv.State = newState
	wv.ErrorState = newState.Error
	if newState.Error {
		wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, myID)
	}
	floor := newState.Floor

	if wv.AllCabOrders == nil {
		wv.AllCabOrders = make(map[string][elev.N_FLOORS]bool)
	}

	if floor < 0 || floor >= elev.N_FLOORS {
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
	if newState.Behaviour == elev.EB_DoorOpen {
		checkFloor = floor
	} else if prevState.Behaviour == elev.EB_DoorOpen {
		checkFloor = prevState.Floor
	}

	if checkFloor < 0 || checkFloor >= elev.N_FLOORS {
		return wv
	}

	upOrder := wv.HallOrders[checkFloor][elev.B_HallUp]
	if upOrder.SyncState == Confirmed &&
		!newState.Requests[checkFloor][elev.B_HallUp] &&
		(prevState.Requests[checkFloor][elev.B_HallUp] || upOrder.OwnerID == myID) {
		upOrder.SyncState = DeleteProposed
		wv.HallOrders[checkFloor][elev.B_HallUp] = upOrder
	}

	downOrder := wv.HallOrders[checkFloor][elev.B_HallDown]
	if downOrder.SyncState == Confirmed &&
		!newState.Requests[checkFloor][elev.B_HallDown] &&
		(prevState.Requests[checkFloor][elev.B_HallDown] || downOrder.OwnerID == myID) {
		downOrder.SyncState = DeleteProposed
		wv.HallOrders[checkFloor][elev.B_HallDown] = downOrder
	}

	return wv
}

// extractRequestsForElevator returns the request matrix for a single elevator:
// confirmed hall orders owned by myID + cab orders for myID.
func extractRequestsForElevator(wv Worldview, myID string) [elev.N_FLOORS][elev.N_BUTTONS]bool {
	var requests [elev.N_FLOORS][elev.N_BUTTONS]bool
	for f := 0; f < elev.N_FLOORS; f++ {
		if wv.HallOrders[f][elev.B_HallUp].SyncState == Confirmed && wv.HallOrders[f][elev.B_HallUp].OwnerID == myID {
			requests[f][elev.B_HallUp] = true
		}
		if wv.HallOrders[f][elev.B_HallDown].SyncState == Confirmed && wv.HallOrders[f][elev.B_HallDown].OwnerID == myID {
			requests[f][elev.B_HallDown] = true
		}
		if wv.AllCabOrders != nil && wv.AllCabOrders[myID][f] {
			requests[f][elev.B_Cab] = true
		}
	}
	return requests
}

func updateOwnerIDsFromAssignment(hallOrders HallOrders, assignment map[string]AssignmentMatrix) HallOrders {
	ho := hallOrders
	for floor := 0; floor < elev.N_FLOORS; floor++ {
		for dir := 0; dir < elev.N_DIRECTIONS; dir++ {
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
	for floor := elev.N_FLOORS - 1; floor >= 0; floor-- {
		fmt.Printf("  Floor %d:\n", floor)
		for dir := 0; dir < elev.N_DIRECTIONS; dir++ {
			order := hallOrders[floor][dir]
			fmt.Printf("    %-4s state=%-14s owner=%q\n", debugHallDirection(dir), debugOrderSyncState(order.SyncState), order.OwnerID)
		}
	}
}

func GoroutineForWorldview(myID string, ch WorldviewChannels) {
	worldviews := make(map[string]Worldview)
	initialWv := Worldview{
		IdElevator:   myID,
		AllCabOrders: map[string][elev.N_FLOORS]bool{myID: {}},
	}
	worldviews[myID] = worldviewInit(myID, initialWv, ch.InitWorldview)

	hasNetwork := true

	sendLights := func() {
		sendLatestWorldview(ch.Lights, copyWorldviews(worldviews)[myID])
	}
	sendToFSM := func() {
		requests := extractRequestsForElevator(worldviews[myID], myID)
		select {
		case ch.ToFSM <- requests:
		default:
			select {
			case <-ch.ToFSM:
			default:
			}
			ch.ToFSM <- requests
		}
	}
	sendToNetwork := func(wv Worldview) { sendLatestWorldview(ch.ToNetwork, wv) }
	sendToSync := func(wvs map[string]Worldview) { sendLatestWorldviewMap(ch.ToSync, wvs) }
	sendToAssigner := func(wvs map[string]Worldview) { sendLatestWorldviewMap(ch.ToAssigner, wvs) }

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
			sendToFSM()

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
	if floor < 0 || floor >= elev.N_FLOORS {
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
	if floor < 0 || floor >= elev.N_FLOORS || dir < 0 || dir >= elev.N_DIRECTIONS {
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
