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

// Special OwnerID values used in the synchronization protocol.
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
type AssignmentMatrix = t.AssignmentMatrix

// WorldviewChannels groups all channels into and out of the worldview goroutine.
type WorldviewChannels struct {
	// Worldview reads from these
	ElevatorState  <-chan t.ElevatorState
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
	ToFSM      chan Worldview
}

// copyWorldviews creates a deep copy of the worldviews map, including AllCabOrders.
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
	localWorldview := myWorldview
	timeout := time.After(1 * time.Second)
	for {
		select {
		case incomingWv := <-initCh:
			if incomingWv.IdElevator == myID {
				continue
			}
			copied := copyWorldviews(map[string]Worldview{incomingWv.IdElevator: incomingWv})[incomingWv.IdElevator]
			localWorldview.HallOrders = copied.HallOrders
			localWorldview.AllCabOrders = copied.AllCabOrders
			return localWorldview

		case <-timeout:
			return localWorldview
		}
	}
}

// _____________________________________________________________________________
// ----------FUNCTIONS FOR RECEIVING AND HANDLING DATA FROM OTHER MODULES--------
// _____________________________________________________________________________

// shouldAcceptSyncOrder determines whether the sync result is valid progress
// and not a stale result that would regress local state.
// shouldAcceptSyncOrder determines whether the sync result represents valid progress.
// The cycle is: None(0) -> Unconfirmed(1) -> Confirmed(2) -> DeleteProposed(3) -> None(0)
func shouldAcceptSyncOrder(localOrder, syncOrder Order) bool {
	if syncOrder.SyncState == localOrder.SyncState {
		return true
	}

		// Forward in the cycle (numerically)
		// Exception: do not reconfirm an order we have already marked as PeerDied
	if syncOrder.SyncState > localOrder.SyncState {
		staleConfirm := localOrder.SyncState == Unconfirmed &&
			localOrder.OwnerID == PeerDied &&
			syncOrder.SyncState == Confirmed &&
			syncOrder.OwnerID != NoOwner
		return !staleConfirm
	}

	// Cycle completion: DeleteProposed -> None
	if localOrder.SyncState == DeleteProposed && syncOrder.SyncState == None {
		return true
	}

	// PeerDied downgrade: Confirmed -> Unconfirmed/PeerDied
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

	for f := 0; f < NumFloors; f++ {
		for d := 0; d < Directions; d++ {
			localOrder := wv.HallOrders[f][d]
			syncOrder := incomingOrders[f][d]

			if !shouldAcceptSyncOrder(localOrder, syncOrder) {
				// Stale sync result; keep the local state
				merged[f][d] = localOrder
				continue
			}

			// Preserve the locally set OwnerID only if sync has no concrete owner.
			// If sync has assigned a concrete owner (for example via conflict resolution), use it.
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

// applyPeerWorldview stores the peer's worldview and updates local state based on it:
// it synchronizes cab orders and downgrades hall orders if the peer is in an error state.
func applyPeerWorldview(worldviews map[string]Worldview, peerWorldview Worldview, myID string) map[string]Worldview {
	worldviews[peerWorldview.IdElevator] = peerWorldview

	wv := worldviews[myID]
	if wv.AllCabOrders == nil {
		wv.AllCabOrders = make(map[string][NumFloors]bool)
	}
	wv.AllCabOrders[peerWorldview.IdElevator] = peerWorldview.AllCabOrders[peerWorldview.IdElevator]
	if peerWorldview.ErrorState {
		wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, peerWorldview.IdElevator)
	}
	worldviews[myID] = wv
	return worldviews
}

// markPeerDeadInHallOrders downgrades Confirmed orders owned by lostId to
// Unconfirmed/PeerDied so that other elevators can take over the order.
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

// updateWorldviewWithElevatorState updates the worldview with a new elevator state from the FSM,
// including the error state, served cab orders, and completed hall orders (setting DeleteProposed).
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

	// Check for served hall orders:
	// Case 1: the FSM is DoorOpen now (normal case - check the current floor)
	// Case 2: the FSM WAS DoorOpen and has just closed the door (catches missed clears)
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

func updateOwnerIDsFromAssignment(hallOrders HallOrders, assignment map[string]AssignmentMatrix) HallOrders {
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

func RunWorldview(myID string, ch WorldviewChannels) {
	worldviews := make(map[string]Worldview)
	initialWv := Worldview{
		IdElevator:   myID,
		AllCabOrders: map[string][NumFloors]bool{myID: {}},
	}
	worldviews[myID] = worldviewInit(myID, initialWv, ch.InitWorldview)

	networkAvailable := true

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

		case peerWorldview := <-ch.PeerWorldview:
			if peerWorldview.IdElevator == myID {
				continue
			}
			worldviews = applyPeerWorldview(worldviews, peerWorldview, myID)
			sendToNetwork(copyWorldviews(worldviews)[myID])
			sendToSync(copyWorldviews(worldviews))

		case newPeerID := <-ch.NewPeer:
			fmt.Printf("[Worldview] Ny peer oppdaget: %s\n", newPeerID)
			if newPeerID == myID {
				networkAvailable = true
				// Restore hallOrders from a known peer on reconnect
				wv := worldviews[myID]
				for id, peerWorldview := range worldviews {
					if id != myID {
						wv.HallOrders = peerWorldview.HallOrders
						break
					}
				}
				worldviews[myID] = wv
			}

		case lostPeerID := <-ch.LostPeer:
			if lostPeerID == myID {
				networkAvailable = false
			}
			worldviews = handleLostPeer(worldviews, myID, lostPeerID)
			sendToNetwork(copyWorldviews(worldviews)[myID])
			sendToSync(copyWorldviews(worldviews))

		case hallBtn := <-ch.HallBtn:
			if networkAvailable {
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

// handleLostPeer marks a lost peer as dead and downgrades its orders to Unconfirmed/PeerDied.
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
