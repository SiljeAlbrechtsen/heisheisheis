package worldview

import (
	fsm "Project/FSM"
	t "Project/types"
	"fmt"

	//"sync"
	"time"
)

//TODO
/*
Teste mergingen
Legge til timer på obstruction
Lage logikk for når en heis kommer tilbake etter mistet internett. Kopiere bare hall orders. Hvordan skal den gjøre det?

*/

//______________________________________________________________________________________________________
//----------------  Structs ----------------------------------------------------------------------------
//______________________________________________________________________________________________________

const (
	Directions = 2
	NumFloors  = 4
)

// Brukes til OwnerID
const (
	PeerDied = "peerDied"
	NoOwner  = ""
)

// type CabOrders [NumFloors]bool // Må vel ikke deklareres først?
type OrderSyncState = t.OrderSyncState

const ( //TODO: Fjerne disse og bruk types direkte
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
	//fmt.Println("hei")
	for {
		select {
		// Hvis den får andre worldvies
		case incomingWv := <-networkToInitCh:
			//Ignorerer seg selv
			if incomingWv.IdElevator == myId {
				continue
			}

			// Koprierer alle cab og hallorders
			myWv.AllCabOrders = incomingWv.AllCabOrders
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

// SYNC sin func
func updateWorldviewFromSync(latestWorldviews map[string]Worldview, inputSyncedHallOrders HallOrders, myID string) map[string]Worldview {
	worldviewsMap := latestWorldviews
	worldview := worldviewsMap[myID]
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

// TODO
// funksjon som legger inn caborders/hallorders inn i din egen worldview. evt samle de sånn at vi kan bruke samme funksjon for de

// Setter state fra confirmet til uncondiremd og ownerID til PeerDied, kjøres når heis dør
func markPeerDeadInHallOrders(hallOrders HallOrders, lostId string) HallOrders {
	ho := hallOrders
	//fmt.Printf("I markPeerDeadInHallOrders \n")
	for i, row := range ho {
		for j := range row {
			order := ho[i][j]

			if order.OwnerID == lostId && order.SyncState == Confirmed {
				order.SyncState = Unconfirmed
				order.OwnerID = PeerDied

			}
			ho[i][j] = order
			//fmt.Printf("Floor %d Dir %d: %+v\n", i, j, order)
		}
	}
	//fmt.Println(":")
	return ho
}

// Endring
func dirToIndex(d fsm.Direction) int {
	if d == fsm.D_Up {
		return 1
	}
	return 0
}

// Mottar elevatorState på channel fra FSM, bruke dette til å oppdatere state og
//
//	ordre i worldview.
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

	if inputStateElevator.Behaviour != fsm.EB_DoorOpen {
		return wv
	}

	// Propose delete only for hall requests this elevator has actually cleared on this floor.
	upOrder := wv.HallOrders[floor][fsm.B_HallUp]
	upWasCleared := (prevState.Requests[floor][fsm.B_HallUp] && !inputStateElevator.Requests[floor][fsm.B_HallUp]) ||
		(upOrder.OwnerID == myID && !inputStateElevator.Requests[floor][fsm.B_HallUp])
	if upOrder.SyncState == Confirmed && upWasCleared {
		upOrder.SyncState = DeleteProposed
		wv.HallOrders[floor][fsm.B_HallUp] = upOrder
	}

	downOrder := wv.HallOrders[floor][fsm.B_HallDown]
	downWasCleared := (prevState.Requests[floor][fsm.B_HallDown] && !inputStateElevator.Requests[floor][fsm.B_HallDown]) ||
		(downOrder.OwnerID == myID && !inputStateElevator.Requests[floor][fsm.B_HallDown])
	if downOrder.SyncState == Confirmed && downWasCleared {
		downOrder.SyncState = DeleteProposed
		wv.HallOrders[floor][fsm.B_HallDown] = downOrder
	}

	return wv
}

// _______________________________________________________________
// ----------------GOROUTINE FOR WORLDVIEW------------------------
// _______________________________________________________________

// Når worldview oppdateres skal sendWorldviewsToOtherModules kjøres

// Endret, lagt til. Den er litt feil tror jeg for den setter bare ownerID til vår egen heis og ikke de ordrene som blir tatt av andre heiser
func updateOwnerIDsFromAssignment(hallOrders HallOrders, assignment map[string][4][3]bool) HallOrders {
	ho := hallOrders
	for floor := 0; floor < NumFloors; floor++ {
		for dir := 0; dir < Directions; dir++ {
			if ho[floor][dir].SyncState == Confirmed {
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

func DebugPrintAllCabOrders(context string, allCabOrders map[string][NumFloors]bool) {
	//fmt.Printf("\n[Worldview] AllCabOrders %s\n", context)
	if len(allCabOrders) == 0 {
		//fmt.Printf("  (tom)\n")
		return
	}
	for id, orders := range allCabOrders {
		fmt.Printf("  elevator=%q  floors: ", id)
		anyActive := false
		for floor, active := range orders {
			if active {
				fmt.Printf("%d ", floor)
				anyActive = true
			}
		}
		if !anyActive {
			fmt.Printf("(ingen)")
		}
		fmt.Println()
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
	hallLightsCh chan HallOrders,
	printHallOrdersReqCh <-chan bool, //ToDO Fjern etter testing

	assignerToWorldviewCh <-chan map[string][4][3]bool,

	worldviewToAssignerCh chan<- map[string]Worldview,
	worldviewToSyncCh chan<- map[string]Worldview,
	worldviewToNetworkCh chan<- Worldview,
	worldviewToFSMCh chan Worldview, //TODO
) {

	worldviewsMap := make(map[string]Worldview)
	myWorldview := worldviewsMap[myID]
	myWorldview.IdElevator = myID
	myWorldview.ErrorState = false
	myWorldview.AllCabOrders = make(map[string][NumFloors]bool)
	myWorldview.AllCabOrders[myID] = [NumFloors]bool{}
	myWorldview = worldviewInit(myID, myWorldview, networkToInitCh)
	worldviewsMap[myID] = myWorldview

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

	sendLatestHallOrders := func(hallOrders HallOrders) {
		select {
		case hallLightsCh <- hallOrders:
		default:
			select {
			case <-hallLightsCh:
			default:
			}
			hallLightsCh <- hallOrders
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

	for {
		select {
		case inputStateElevator := <-elevatorToWorldviewCh:
			myWorldview = worldviewsMap[myID] // A-La til denne for å sikre at vi har siste versjon av worldview før vi oppdaterer den
			myWorldview = updateWorldviewWithElevatorState(myWorldview, inputStateElevator, myID)
			if myWorldview.AllCabOrders == nil {
				myWorldview.AllCabOrders = make(map[string][NumFloors]bool)
			}

			if inputStateElevator.Error {
				myWorldview.ErrorState = true
				myWorldview.HallOrders = markPeerDeadInHallOrders(myWorldview.HallOrders, myID)
			} else {
				myWorldview.ErrorState = false
			}

			worldviewsMap[myID] = myWorldview
			sendLatestHallOrders(myWorldview.HallOrders)
			worldviewToNetworkCh <- copyMap(worldviewsMap)[myID]
			worldviewToSyncCh <- copyMap(worldviewsMap)

		case inputSyncedHallOrders := <-syncToWorldviewCh:
			worldviewsMap = updateWorldviewFromSync(worldviewsMap, inputSyncedHallOrders, myID)
			myWorldview = worldviewsMap[myID]
			worldviewToAssignerCh <- copyMap(worldviewsMap)
			sendLatestHallOrders(myWorldview.HallOrders)
			worldviewToNetworkCh <- copyMap(worldviewsMap)[myID]

		case inputPeerWorldview := <-networkToWorldviewCh:
			if inputPeerWorldview.IdElevator == myID {
				continue
			}
			worldviewsMap = updatePeerWorldviewFromNetwork(worldviewsMap, inputPeerWorldview)
			myWorldview = worldviewsMap[myID]
			if myWorldview.AllCabOrders == nil {
				myWorldview.AllCabOrders = make(map[string][NumFloors]bool)
			}

			myWorldview.AllCabOrders[inputPeerWorldview.IdElevator] = inputPeerWorldview.AllCabOrders[inputPeerWorldview.IdElevator]

			if inputPeerWorldview.ErrorState {
				myWorldview.HallOrders = markPeerDeadInHallOrders(myWorldview.HallOrders, inputPeerWorldview.IdElevator)
			}

			worldviewsMap[myID] = myWorldview
			//DebugPrintAllCabOrders(fmt.Sprintf("etter peer-oppdatering fra %q", inputPeerWorldview.IdElevator), myWorldview.AllCabOrders)
			worldviewToSyncCh <- copyMap(worldviewsMap)

		case newPeer := <-newPeerIdCh:
			fmt.Printf("[Worldview] Ny peer oppdaget: %s\n", newPeer)

		case inputDeadPeer := <-lostPeerIdCh:
			//fmt.Printf("[Worldview] Peer tapt: %s\n", inputDeadPeer)
			worldviewsMap = HandleLostPeer(worldviewsMap, myID, inputDeadPeer)
			myWorldview = worldviewsMap[myID]
			worldviewToSyncCh <- copyMap(worldviewsMap)

		case inputHallBtn := <-hallBtnCh:
			myWorldview = worldviewsMap[myID]
			myWorldview = addNewHallOrder(myWorldview, inputHallBtn)
			worldviewsMap[myID] = myWorldview
			sendLatestHallOrders(myWorldview.HallOrders)
			worldviewToNetworkCh <- copyMap(worldviewsMap)[myID]
			worldviewToSyncCh <- copyMap(worldviewsMap)

		case inputCabBtn := <-cabBtnCh:
			myWorldview = worldviewsMap[myID]
			myWorldview = addNewCabOrder(myWorldview, inputCabBtn, myID)

			worldviewsMap[myID] = myWorldview
			//DebugPrintAllCabOrders(fmt.Sprintf("etter cab-knapp floor=%d", inputCabBtn), myWorldview.AllCabOrders)
			sendLatestHallOrders(myWorldview.HallOrders)
			worldviewToNetworkCh <- copyMap(worldviewsMap)[myID]
			worldviewToSyncCh <- copyMap(worldviewsMap)

		case inputAssignment := <-assignerToWorldviewCh:
			myWorldview = worldviewsMap[myID]
			debugPrintHallOrders("before assignment", myWorldview.HallOrders) // TO DO: FJERN
			myWorldview.HallOrders = updateOwnerIDsFromAssignment(myWorldview.HallOrders, inputAssignment)
			debugPrintHallOrders("after assignment", myWorldview.HallOrders) // TO DO: FJERN

			worldviewsMap[myID] = myWorldview
			sendLatestHallOrders(myWorldview.HallOrders)
			worldviewToNetworkCh <- copyMap(worldviewsMap)[myID]
			sendLatestWorldviewToFSM(copyMap(worldviewsMap)[myID]) //TODO

		case <-printHallOrdersReqCh:
			myWorldview = worldviewsMap[myID]
			debugPrintHallOrders("stop button worldview", myWorldview.HallOrders)
		}

	}
}

// Vi har gjort det slik at alt som skal til assigner går først innom sync.

//___________________________________________________________________________
//------FUNKSJONER FOR Å SENDE KOPIERT WORLDVIEW TIL ANDRE MODULER-----------
//___________________________________________________________________________

// Tar inn map, setter den døde noden sin state til død og oppdaterer ordre til død node
func HandleLostPeer(latestWorldviews map[string]Worldview, myID string, lostID string) map[string]Worldview {
	//fmt.Printf("[Worldview] HandleLostPeer: lostID=%s myID=%s\n", lostID, myID)
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
	wv := worldview

	cabOrders := wv.AllCabOrders[myID]
	cabOrders[inputCabBtn] = true
	wv.AllCabOrders[myID] = cabOrders

	return wv
}

func addNewHallOrder(worldview Worldview, inputHallBtn [2]int) Worldview {
	wv := worldview

	floor := inputHallBtn[0]
	dir := inputHallBtn[1]

	order := wv.HallOrders[floor][dir]

	order.SyncState = Unconfirmed
	order.OwnerID = NoOwner

	wv.HallOrders[floor][dir] = order

	return wv
}
