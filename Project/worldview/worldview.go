package worldview

import (
	fsm "Project/FSM"
	"time"
)

/*
// Endret: Har satt ownerID som string

TODO
Lage setOwnerId.
Må LEGGE TIL Case fra channel fra assigner som kjører funk
Assigner kjøres kontinuerlig, så vil den ta opp hele worldview eller går det bra
Evt bare sende når noe endres.

  1. Sync sørger for at alle er enige om at en ordre er Confirmed — men setter OwnerID = NoOwner
  2. Assigner kjøres lokalt på hver heis med samme input → produserer samme resultat
  3. updateOwnerIDsFromAssignment setter OwnerID basert på assignerens resultat
  4. Worldview broadcastes → alle ender opp med samme OwnerID

BUG? Endret slik at sync ikke setter ownerID til noOwner for da blir den nullet hele tiden.


// Er det noe som setter ordrene til død heis til unconfirmed og peer dead ?

Hvis vi har flere mottakere på en channel vil bare den som var først klar, motta verdien.
De fungerer ved at de "leser av en jobbkø" elns

  1. myWorldview.IdElevator ble aldri satt til myID
  2. assignerToWordviewCh — ingen leste fra den (deadlock). Løst ved å koble den til worldview og bruke den til å sette OwnerID via updateOwnerIDsFromAssignment
  3. FindFloorFromRequest returnerte 0 i stedet for -1 når ingen ordre
  4. Sync overskrev OwnerID med NoOwner ved Confirmed-overgang
  5. peerUpdateCh hadde to lesere, så noen peer-death events ble tapt


  FSM beveger seg (leser ikke requests)
      → Assigner får ny worldview, regner ut, sender på assignerToFsmCh
      → Ingen leser assignerToFsmCh → Assigner blokkerer
      → Worldview prøver å sende ny worldview til assigner
      → Assigner kan ikke motta (blokkert) → Worldview blokkerer
      → FSM prøver å sende etasjeoppdatering til worldview
      → Worldview kan ikke motta (blokkert) → FSM blokkerer
      → DEADLOCK
Fikset med buffret channel


  Bug: elevio.SetMotorDirection kalles aldri i FSM2
FSM/FSM.go — elevio.SetMotorDirection aldri kalt i FSM2
  UpdateDirection oppdaterer bare intern state og sender til worldview — den starter ikke motoren fysisk. Lagt til SetMotorDirection-kall på fire steder: når ingen ordre, når retning settes fra
  requests-casen, ved ankomst, og når retning settes fra floorTicker-casen.
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
type OrderSyncState int

const (
	None OrderSyncState = iota
	// En heis setter til unconfirmed. Når de andre er enige så setter de til confirmed.
	Unconfirmed
	// I confirmed så får den ownerID. Den blir assigned.
	Confirmed
	DeleteProposed
)

type Order struct {
	SyncState OrderSyncState
	OwnerID   string
}

type HallOrders [NumFloors][Directions]Order

// Struct for egen worldview
type Worldview struct {
	IdElevator   string
	HallOrders   HallOrders
	State        fsm.ElevatorState
	MycabOrders  [NumFloors]bool // En liste med true or false for hver eneste etasje å trykke inn
	AllCabOrders map[string][NumFloors]bool
	ErrorState   bool
}

func worldviewInit(myId string, myWorldview Worldview, networkToWorldviewCh <-chan Worldview) Worldview {
	myWv := myWorldview
	timeout := time.After(5 * time.Second)

	for {
		select {
		// Hvis den får andre worldvies
		case incomingWv := <-networkToWorldviewCh:
			//Ignorerer seg selv
			  if incomingWv.IdElevator == myId {
                continue
            }
			
			// Koprierer alle cab og hallorders
			myWv.AllCabOrders = incomingWv.AllCabOrders
			myWv.HallOrders = incomingWv.HallOrders

		   // henter egne cabOrders hvis de finnes i AllCaborders
            if caborders, exists := incomingWv.AllCabOrders[myId]; exists {
                myWv.MycabOrders = caborders
            }
		
			return myWv // ferdig init
		// Hvis de ikke får noe fra andre
		case <- timeout:
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
func updateWorldviewWithElevatorState(worldview Worldview, inputStateElevator fsm.ElevatorState) Worldview {
	wv := worldview
	wv.State = inputStateElevator
	floor := inputStateElevator.Floor

	if floor < 0 || floor >= NumFloors {
		return wv
	}

	if wv.MycabOrders[floor] {
		wv.MycabOrders[floor] = false
	}

	// Når heisen betjener en etasje (dør åpen), sett alle hall-ordre på etasjen til DeleteProposed
	if inputStateElevator.Behaviour == fsm.EB_DoorOpen {
		for dir := 0; dir < Directions; dir++ {
			if wv.HallOrders[floor][dir].SyncState == Confirmed {
				wv.HallOrders[floor][dir].SyncState = DeleteProposed
			}
		}
		return wv
	}

	if inputStateElevator.Dirn == fsm.D_Stop {
		return wv
	}

	dir := dirToIndex(inputStateElevator.Dirn)
	if wv.HallOrders[floor][dir].SyncState == Confirmed {
		wv.HallOrders[floor][dir].SyncState = DeleteProposed
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

func GoroutineForWorldview(
	myID string,
	elevatorToWorldviewCh <-chan fsm.ElevatorState,
	syncToWorldviewCh <-chan HallOrders,
	networkToWorldviewCh <-chan Worldview,

	lostPeerIdCh <-chan string,
	newPeerIdCh <-chan string,
	cabBtnCh <-chan int,
	hallBtnCh <-chan [2]int,

	assignerToWorldviewCh <-chan map[string][4][3]bool,

	worldviewToAssignerCh chan<- map[string]Worldview,
	worldviewToSyncCh chan<- map[string]Worldview,
	worldviewToNetworkCh chan<- Worldview,
) {
	// NB!!!
	// Pass på at channelsene bare sender inn når det skjer en endring, slik at de ikke blokkerer
	// TODO må også skaffe logikk med når peer er død at den ikke tas med i beregninger i sync og assigner. Egen activ state i worldview
	worldviewsMap := make(map[string]Worldview)
	myWorldview := worldviewsMap[myID]
	myWorldview.IdElevator = myID
	worldviewsMap[myID] = myWorldview

	copyMap := func(m map[string]Worldview) map[string]Worldview {
		c := make(map[string]Worldview, len(m))
		for k, v := range m {
			c[k] = v
		}
		return c
	}

	for {
		select {
		case init := <-newPeerIdCh:
			if init == myID {
				myWorldview = worldviewInit(myID, myWorldview, networkToWorldviewCh)
				worldviewsMap[myID] = myWorldview
			}

		case inputStateElevator := <-elevatorToWorldviewCh:
			myWorldview = updateWorldviewWithElevatorState(myWorldview, inputStateElevator)
			worldviewsMap[myID] = myWorldview
			worldviewToNetworkCh <- worldviewsMap[myID]
			worldviewToSyncCh <- copyMap(worldviewsMap)

		case inputSyncedHallOrders := <-syncToWorldviewCh:
			worldviewsMap = updateWorldviewFromSync(worldviewsMap, inputSyncedHallOrders, myID)
			myWorldview = worldviewsMap[myID]
			worldviewToNetworkCh <- worldviewsMap[myID]
			worldviewToAssignerCh <- copyMap(worldviewsMap)

		case inputPeerWorldview := <-networkToWorldviewCh:
			worldviewsMap = updatePeerWorldviewFromNetwork(worldviewsMap, inputPeerWorldview)
			worldviewToSyncCh <- copyMap(worldviewsMap)

		case inputDeadPeer := <-lostPeerIdCh:
			worldviewsMap = HandleLostPeer(worldviewsMap, myID, inputDeadPeer)
			worldviewToSyncCh <- copyMap(worldviewsMap)

		case inputHallBtn := <-hallBtnCh:
			myWorldview = addNewHallOrder(myWorldview, inputHallBtn)
			worldviewsMap[myID] = myWorldview
			worldviewToNetworkCh <- worldviewsMap[myID]
			worldviewToSyncCh <- copyMap(worldviewsMap)

		case inputCabBtn := <-cabBtnCh:
			myWorldview = addNewCabOrder(myWorldview, inputCabBtn)
			worldviewsMap[myID] = myWorldview
			worldviewToNetworkCh <- worldviewsMap[myID]
			worldviewToAssignerCh <- copyMap(worldviewsMap)

		case inputAssignment := <-assignerToWorldviewCh:
			myWorldview.HallOrders = updateOwnerIDsFromAssignment(myWorldview.HallOrders, inputAssignment)
			worldviewsMap[myID] = myWorldview
		}

	}
}

// Vi har gjort det slik at alt som skal til assigner går først innom sync.

//___________________________________________________________________________
//------FUNKSJONER FOR Å SENDE KOPIERT WORLDVIEW TIL ANDRE MODULER-----------
//___________________________________________________________________________

/*
type HallOrdersPublic [NumFloors][Directions]Order

type TransferWorldview struct {
	IdElevator  string
	HallOrders  HallOrders
	State       fsm.ElevatorState
	MycabOrders [NumFloors]bool
}

func copyWorldview(worldview Worldview) TransferWorldview {
	return TransferWorldview{
		IdElevator:  worldview.IdElevator,
		HallOrders:  worldview.HallOrders,
		State:       worldview.State,
		MycabOrders: worldview.MycabOrders,
	}
}

func copyWorldviews(latestWorldviews map[string]Worldview) map[string]TransferWorldview {
	copied := make(map[string]TransferWorldview, len(latestWorldviews))
	for id, worldview := range latestWorldviews {
		copied[id] = copyWorldview(worldview)
	}
	return copied
}
*/

// Tar inn map, setter den døde noden sin state til død og oppdaterer ordre til død node
func HandleLostPeer(latestWorldviews map[string]Worldview, myID string, lostID string) map[string]Worldview {
	lwv := latestWorldviews
	lostWorldview := lwv[lostID]
	lostWorldview.ErrorState = true
	lwv[lostID] = lostWorldview

	wv := lwv[myID]
	wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, lostID)

	lwv[myID] = wv

	return lwv
}

func addNewCabOrder(worldview Worldview, inputCabBtn int) Worldview {
	wv := worldview

	wv.MycabOrders[inputCabBtn] = true

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
