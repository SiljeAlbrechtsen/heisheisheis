package worldview

import (
	"strconv"
	"Project/FSM"

)

/*
TODO
Lage setOwnerId. 
Må LEGGE TIL Case fra channel fra assigner som kjører funk
Assigner kjøres kontinuerlig, så vil den ta opp hele worldview eller går det bra
Evt bare sende når noe endres. 

*/


//______________________________________________________________________________________________________
//----------------  Structs ----------------------------------------------------------------------------
//______________________________________________________________________________________________________

const (
	Directions = 2
	NumFloors  = 3
)

// Brukes til OwnerID
const (
	PeerDied = -1
	NoOwner  = -2
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
	OwnerID   int
}

type HallOrders [NumFloors][Directions]Order

// Struct for egen worldview
type Worldview struct {
	IdElevator  string
	HallOrders  HallOrders
	State       fsm.ElevatorState  
	MycabOrders [NumFloors]bool // En liste med true or false for hver eneste etasje å trykke inn
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
func updatePeerWorldviewFromNetwork(latestWorldviews map[string]Worldview, inputPeerWorldview Worldview) map[string]Worldview{
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

	for i, row := range ho{
		for j := range row {
			order := ho[i][j]	

	
			if strconv.Itoa(order.OwnerID) == lostId && order.SyncState == Confirmed {
				order.SyncState = Unconfirmed
				order.OwnerID = PeerDied
				
				}	
			ho[i][j] = order
			}
		}
	return ho	
}

// Mottar elevatorState på channel fra FSM, bruke dette til å oppdatere state og
//  ordre i worldview.
func updateWorldviewWithElevatorState(worldview Worldview, inputStateElevator fsm.ElevatorState) Worldview {
    wv := worldview
    wv.State = inputStateElevator
    floor := inputStateElevator.Floor
    dir := inputStateElevator.Dirn

    if strconv.Itoa(wv.HallOrders[floor][dir].OwnerID) == wv.IdElevator {
        if wv.HallOrders[floor][dir].SyncState == Confirmed {
            wv.HallOrders[floor][dir].SyncState = DeleteProposed
        }
    }
    if wv.MycabOrders[floor] == true {
        wv.MycabOrders[floor] = false
    }
    return wv
}




// _______________________________________________________________
// ----------------GOROUTINE FOR WORLDVIEW------------------------
// _______________________________________________________________

// Når worldview oppdateres skal sendWorldviewsToOtherModules kjøres


func GoroutineForWorldview(
	myID 						  string,
	elevatorToWorldviewCh   <-chan fsm.ElevatorState,
	syncToWorldviewCh       <-chan HallOrders,
	networkToWorldviewCh    <-chan Worldview,

	lostPeerIdCh    		<-chan string,
	cabBtnCh 				<-chan int,
	hallBtnCh   			<-chan [2]int,
	
	worldviewToAssignerCh   chan<- map[string]Worldview,
	worldviewToSyncCh       chan<- map[string]Worldview,
	worldviewToNetworkCh    chan<- Worldview,
	) {
// NB!!!
// Pass på at channelsene bare sender inn når det skjer en endring, slik at de ikke blokkerer
	// TODO må også skaffe logikk med når peer er død at den ikke tas med i beregninger i sync og assigner. Egen activ state i worldview
	worldviewsMap := make(map[string]Worldview)
	myWorldview := worldviewsMap[myID]

	for {
		select {
		
		// Får inn endring i stateElevator fra FSM. Oppdaterer worldview med ny state og oppdaterer fullførte ordre
		case inputStateElevator := <-elevatorToWorldviewCh:
			myWorldview = updateWorldviewWithElevatorState(myWorldview, inputStateElevator) // ingrid hjelp
			worldviewsMap[myID] = myWorldview
			worldviewToNetworkCh <- worldviewsMap[myID]
			worldviewToSyncCh <- worldviewsMap
			

	// Får inn syncet hallorders fra sync. Den må da oppdatere worldview også sende den oppdaterte til andre moduler
		case inputSyncedHallOrders := <-syncToWorldviewCh:
			worldviewsMap = updateWorldviewFromSync(worldviewsMap, inputSyncedHallOrders, myID)
			// TODO Trenger ikke sende til sync
			worldviewToNetworkCh <- worldviewsMap[myID]
			worldviewToAssignerCh <- worldviewsMap // Sender bare til Assigner her?

		
		// Får inn en peers worldview. Må Oppdatere map og sende til andre moduler
		case inputPeerWorldview := <-networkToWorldviewCh:
			worldviewsMap = updatePeerWorldviewFromNetwork(worldviewsMap, inputPeerWorldview)
			// TODO Bare Sync trenger denne info
			worldviewToSyncCh <- worldviewsMap

		// Får inn at en peer er død
		case inputDeadPeer := <-lostPeerIdCh:
			// TODO: Hvordan sette node død?
			// peerdead funksjonen må kjøres her et sted. 
			worldviewsMap = HandleLostPeer(worldviewsMap, myID, inputDeadPeer)
			worldviewToSyncCh <- worldviewsMap
			// Sync + assigner 


		case inputHallBtn := <-hallBtnCh:
			myWorldview = addNewHallOrder(myWorldview, inputHallBtn)
			worldviewsMap[myID] = myWorldview
			
			worldviewToNetworkCh <- worldviewsMap[myID]
			worldviewToSyncCh <- worldviewsMap
			// TODO Network, Sync, Assigner

		case inputCabBtn := <-cabBtnCh:
			myWorldview = addNewCabOrder(myWorldview, inputCabBtn)
			worldviewsMap[myID] = myWorldview

			worldviewToNetworkCh <- worldviewsMap[myID]
			worldviewToAssignerCh <- worldviewsMap
				
		// TODO Network, Sync, Assigner

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
func HandleLostPeer(latestWorldviews map[string]Worldview, myID string, lostID string) map[string]Worldview{
	lwv := latestWorldviews
	///lwv[lostID].state = dead  ???
    wv := lwv[myID]

    wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, lostID)
 
    lwv[myID] = wv

	return lwv
}

func addNewCabOrder(worldview Worldview, inputCabBtn int) Worldview{
	wv := worldview

	wv.MycabOrders[inputCabBtn] = true

	return wv
}

func addNewHallOrder(worldview 	Worldview, inputHallBtn [2]int) Worldview {
	wv := worldview

	floor := inputHallBtn[0]
	dir := inputHallBtn[1]

	order := wv.HallOrders[floor][dir]

	order.SyncState = Unconfirmed
	order.OwnerID = NoOwner

	wv.HallOrders[floor][dir] = order

	return wv
}

