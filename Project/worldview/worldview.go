package worldview

/*
Fullført:
Nå sender worldview til network, assigner og sync. 
Laget sync motta funksjon
laget sync sende funk i sync modul
laget go-routine i sync
Fikse goroutine i assigner

Silje: liste på todo
- fikse goroutine i worldview
- få med alle endringer fra elevator
- LEGG INN CHANNEL I MAIN
- hva som skjer når en peer dør

- lage nettwork motta funk (ingrig)

TODO: Ordne sånn at alle channels har samme navn
*/



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
Ligger hall_request_assigner i riktig mappe?

Testing:
Finne ut hva vi burde teste og hvordan?
Teste en og en Go-routine? Da kan også noen feilsøke, mens andre fortsetter å kode

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





// La de inn i samme funksjon siden de skal kjøres samtidig. Skal kjøres når worldview oppdateres
func sendWorldviewsToOtherModules(
	latestWorldviews map[int]Worldview, 
	updatedWorldviewToNetworkCh chan<- map[string]TransferWorldview, 
	updatedWorldviewToAssignerCh chan<- map[int]TransferWorldview, 
	updatedWorldviewToSyncCh chan<- map[int]TransferWorldview, 
	myID int)  
	{
	updatedWorldviewToNetworkCh <- copyWorldviewsStringKey(latestWorldviews, myID)  
	updatedWorldviewToAssignerCh <- copyWorldviews(latestWorldviews)
	updatedWorldviewToSyncCh <- copyWorldviews(latestWorldviews)
}



// _____________________________________________________________________________
// ----------FUNKSJONER FOR Å TA IMOT OG HÅNDTERE DATA FRA ANDRE MODULER--------
// _____________________________________________________________________________

// Vi får inn data fra sync og network

// SYNC sin func
func updateWorldviewFromSync(latestWorldviews map[int]Worldview, orders hallOrders, myID int) {
	latestWorldviews[myID].hallOrders = orders
}

func updatePeer	WorldviewFromNetwork(worldview Worldview, myID int) { // Ingrid
	/*
	Får inn en worldview. Skal bruke IDen dens til å legge det inn i mappet. 
	Skal også merke om en peer er død? Evt sende det på en annen channel
	*/
}




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

// _______________________________________________________________
// ----------------GOROUTINE FOR WORLDVIEW------------------------
// _______________________________________________________________

// Når worldview oppdateres skal sendWorldviewsToOtherModules kjøres


func GoroutineForWorldview(
	myID 						  int,
	elevatorToWorldviewCh  <-chan StateElevator,
	syncToWorldviewCh      <-chan HallOrders,
	networkToWorldviewCh   <-chan wordlview,
	// TODO
	networkToWorldviewNewPeerCh    <-chan string,
	networkToWorldviewDeadPeerCh   <-chan string, 

	worldviewToAssignerCh   chan<- map[int]Worldview
	worldviewToSyncCh       chan<- map[int]Worldview
	worldviewToNetworkCh    chan<- map[string]TransferWorldview) 
	{ 

	
	worldviewsMap := make(map[int]Worldview)

	for {
		select
	
	// Får inn endring på state elevator. Hva skal skje da?
	case: inputStateElevator := <-elevatorToWorldviewCh
		updateWorldviewWithElevatorState() // ingrid hjelp
		sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
	
		// Får inn syncet hallorders fra sync. Den må da oppdatere får worldview også sende den oppdaterte til andre moduler
	case: inputHallOrders := <-syncToWorldviewCh
		updateWorldviewFromSync(worldviewsMap, inputHallOrders, myID)
		sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
	
		// Får inn en peers worldview. Må Oppdatere map og sende til andre moduler
	case: inputPeerWorldview := <-networkToWorldviewCh
		// Her må updatePeerWorldview ligge
		sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
	
		// Får inn at en peer lever
	case: inputNewPeer := <-networkToWorldviewNewPeerCh
		// Må kjøre en funksjon
		sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
	
		// Får inn at en peer er død
	case: inputDeadPeer := <-networkToWorldviewDeadPeerCh
		// Må kjøre en funksjon
		// peerdead funksjonen må kjøres her et sted. 
		sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
	} 
}

//___________________________________________________________________________
//------FUNKSJONER FOR Å SENDE KOPIERT WORLDVIEW TIL ANDRE MODULER-----------
//___________________________________________________________________________

type hallOrdersPublic [NumFloors][Directions]Order

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

// Siden network opererer med string key, må jeg også ha en funksjon for det. Den returnerer også bare vårt worldview
// Ikke bra cohesion?
func copyOneWorldviewStringKey(latestWorldviews map[int]Worldview, myID int) map[string]TransferLatestWorldviews {
    copied := make(map[string]TransferLatestWorldviews, 1)
    copied[strconv.Itoa(myID)] = latestWorldviews[myID].copyWorldview()
    return copied
}
