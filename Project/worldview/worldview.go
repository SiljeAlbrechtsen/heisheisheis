package worldview

/*
TODO:
Skrive GO-routines for assigner, worldview, sync
Finne ut hvordan håndtere data som kommer fra FSM - Ingrid
Finne ut hvordan håndtere data som kommer fra Network
- Når peer dør.
- Når vi får inn heartbeat. 
Finne ut hvordan man kan ha network alene. Altså at den ikke er main. Go-routine
// Finne ut hvordan vi gjør det når vi bare skal ha en worldview. Evt sette myID som en global const variabel som vi bruker som indeks.
// Når vi starter programmet må vi legge inn vår worldview som ID_0 elns
Være sikker på: håndterer vi button light contract? Når lys skrus på, er alle enige?
Go-routine for sync kjøres hver gang den får inn noe på channel
Samme med worldview

Testing:
Finne ut hva vi burde teste og hvordan?

EnkelHeisLogikk: (Alexsey)
Modularisere koden og strukturere. Forslag: elevator – selve heistilstanden (etasje, retning, dør åpen/lukket, motor). controller / fsm – logikken som bestemmer hva heisen skal gjøre basert på tilstand og bestillinger.
Forstå den
Sende ElevatorState til Worldview Channel elevatorStateCh
Motta matrise fra assigner via channel AssignedRequestsCh
Finne ut hva den skal gjøre med matrisen. Hvordan heisen blir styrt av denne matrisen?
Skrive Go-routines. minst en for FSM og en for driver?
Være sikker på: håndterer vi button light contract?
*/

//______________________________________________________________________________________________________
//----------------  Structs ----------------------------------------------------------------------------
//______________________________________________________________________________________________________

const (
	Directions = 1
	NumFloors  = 3
)

// Brukes til OwnerID
const (
	peerDied = -1
	noOwner  = -2
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
	syncState OrderSyncState
	ownerID   int
}

type HallOrders [NumFloors][Directions]Order

// Struct for egen worldview
type Worldview struct {
	idElevator  int
	hallOrders  HallOrders
	state       StateElevator   // TODO: Må hente type fra fsm
	mycabOrders [NumFloors]bool // En liste med true or false for hver eneste etasje å trykke inn
}

// Struct der alle sine worldviews
// type MergedWorldviews struct {
//	Elevators map[ElevID]ElevState
//}

//____________________________________________________________________________________________________________________
//---------------------- CHANNELS ------------------------------------------------------------------------------------
//____________________________________________________________________________________________________________________


/*
Inn: elevatorState fra FSM
     worldviews fra andre peers fra Network
     oppdaterte orders i hallOrders fra Assigner

Ut: rå worldview-map til sync
    order-lister til Assigner
	nye endringer på nettverk
*/      


// TODO

// funksjon som legger inn caborders/hallorders inn i din egen worldview. evt samle de sånn at vi kan bruke samme funksjon for de


// Setter state fra confirmet til uncondiremd og ownerID til peerDied, kjøres når heis dør
func markPeerDead(order Order) Order {
	if order.orderSyncState == Confirmed {
		order.orderSyncState = Unconfirmed
	}
	order.ownerID = peerDied
	return order
}

//___________________________________________________________________________
//------FUNKSJONER FOR Å SENDE KOPIERT WORLDVIEW TIL ANDRE MODULER-----------
//___________________________________________________________________________

type TransferWorldview struct {
	IdElevator  int
	HallOrders  HallOrders
	State       StateElevator   // TODO: Må hente type fra fsm
	MycabOrders [NumFloors]bool // En liste med true or false for hver eneste etasje å trykke inn
}

func copyWorldview(worldview Worldview) TransferWorldview {
	return TransferWorldview{
		IdElevator:   worldview.idElevator,
		HallOrders:   worldview.hallOrders,
		State:        worldview.state,
		MycabOrders:  worldview.mycabOrders,
	}
}

// TODO: Er det bedre praksis å lage mappet lokalt i funksjonen eller globalt?
// Kopierer worldview inn i nytt map som skal sendes til andre moduler
func copyWorldviews(latestWorldviews map[int]Worldview) map[int]TransferLatestWorldviews {
    copied := make(map[int]TransferLatestWorldviews, len(latestWorldviews))
    for id, worldview := range latestWorldviews {
        copied[id] = worldview.copyWorldview()
    }
    return copied
}

// La de inn i samme funksjon siden de skal kjøres samtidig. 
func sendWorldviewsToOtherModules(latestWorldviews map[int]Worldview, ch chan<- map[int]TransferWorldview, , updatedWorldviewToNetworkCh chan<- map[int]TransferWorldview, updatedWorldviewToAssignerCh chan<- map[int]TransferWorldview, updatedWorldviewToSyncCh chan<- map[int]TransferWorldview)  {
	updatedWorldviewToNetworkCh <- copyWorldviews(latestWorldviews)
	updatedWorldviewToAssignerCh <- copyWorldviews(latestWorldviews)
	updatedWorldviewToSyncCh <- copyWorldviews(latestWorldview)
}

// Mottar elevatorState på channel fra FSM, bruke dette til å oppdatere worldview med data.
func updateWorldviewWithElevatorState(worldview Worldview, elevatorStateCh <-chan StateElevator) Worldview {
    wv := worldview
    elevatorState := <-elevatorStateCh
    wv.state = elevatorState
    floor := elevatorState.floor
    dir := elevatorState.dir

    if wv.hallOrders[floor][dir].ownerID == wv.idElevator {
        if wv.hallOrders[floor][dir].syncState == Confirmed {
            wv.hallOrders[floor][dir].syncState = DeleteProposed
        }
    }
    if wv.mycabOrders[floor] == true {
        wv.mycabOrders[floor] = false
    }
    return wv
}

