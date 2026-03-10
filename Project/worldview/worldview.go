package worldview

/*
Fullført:
Nå sender worldview til network, assigner og sync. 
Laget sync motta funksjon
laget sync sende funk i sync modul
laget go-routine i sync
Fikse goroutine i assigner
- fikse goroutine i worldview

Silje: liste på todo
- få med alle endringer fra elevator
- LEGG INN CHANNEL I MAIN


- hva som skjer når en peer dør
- lage nettwork motta funk (ingrig)

TODO: Ordne sånn at alle channels har samme navn
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

//____________________________________________________________________________________________________________________
//---------------------- CHANNELS -------------------------------	updatedWorldviewToNetworkCh <- copyWorldviews(latestWorldviews)-----------------------------------------------------
//____________________________________________________________________________________________________________________    


// La de inn i samme funksjon siden de skal kjøres samtidig. Skal kjøres når worldview oppdateres
func sendWorldviewsToOtherModules(
	latestWorldviews map[int]Worldview, 
	updatedWorldviewToNetworkCh chan<- map[string]TransferWorldview, 
	updatedWorldviewToAssignerCh chan<- map[int]TransferWorldview, 
	updatedWorldviewToSyncCh chan<- map[int]TransferWorldview, 
	myID int)  {
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

func updatePeerWorldviewFromNetwork(worldview Worldview, myID int) { // Ingrid
	/*
	Får inn en worldview. Skal bruke IDen dens til å legge det inn i mappet. 
	Skal også merke om en peer er død? Evt sende det på en annen channel
	*/
}


// TODO
// funksjon som legger inn caborders/hallorders inn i din egen worldview. evt samle de sånn at vi kan bruke samme funksjon for de

// Setter state fra confirmet til uncondiremd og ownerID til peerDied, kjøres når heis dør
func markPeerDeadInHallOrders(hallOrders Hallorders) Hallorder {
	ho := hallOrders

	for i, row := range ho{
		for j := range row {
			order := ho[i][j]	
			if order.orderSyncState == Confirmed {
				order.orderSyncState = Unconfirmed
				}

				order.ownerID = peerDied
				
			ho[i][j] = order
			}
		}
	return ho	
}


// Mottar elevatorState på channel fra FSM, bruke dette til å oppdatere worldview med data.
func updateWorldviewWithElevatorState(worldview Worldview, elevatorToWorldviewCh <-chan StateElevator) Worldview {
    wv := worldview
    elevatorState := <-elevatorToWorldviewCh
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
	networkToWorldviewCh   <-chan Wordlview,
	// TODO
	newPeerIdCh   <-chan string, 
	lostPeerIdCh    <-chan string,

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
	case: inputNewPeer := <-newPeerIdCh
		// Må kjøre en funksjon

		sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
	
		// Får inn at en peer er død
	case: inputDeadPeer := <-lostPeerIdCh
		// Må kjøre en funksjon
		// peerdead funksjonen må kjøres her et sted. 
		HandleLostPeers(worldviewsMap, inputDeadPeer)
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


func HandleLostPeers(latestWorldviews map[int]Worldview, lostPeerId string){
	for {
		lostID := lostPeerId
		SetNodeDead(latestWorldviews, lostID)
	}
}

func SetNodeDead(latestWorldviews map[int]Worldview, id string){
	lwv := latestWorldviews
	//TODO: sette elevator til dead
	for _, wv := range lwv{
		for _, ho := range wv.hallOrders {
			wv.hallOrders = markPeerDeadInHallOrders(vw.hallOrders)
		}
	}
}

func addNewCabOrder(wordview Worldview, cabButtonCh chan int) {
	wv := worldview
	cabBtn := <- cabButtonCh

	wv.cabOrders[cabBtn] = true
}

func addNewHallOrder(wordview, hallButtonCh chan [2]int) {
	wv := worldview
	hallBtn := <- hallButtonCh

	floor := hallBtn[0]
	dir := hallbtn[1]

	order := wv.hallOrders[floor][dir]

	order.syncState = Unconfirmed
	order.ownerID = noOwner

}

