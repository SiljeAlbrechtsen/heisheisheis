package worldview

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
	myID int,
	latestWorldviews map[int]Worldview, 
	updatedWorldviewToAssignerCh  chan<- map[int]TransferWorldview, 
	updatedWorldviewToNetworkCh   chan<- map[string]TransferWorldview, 
	updatedWorldviewToSyncCh      chan<- map[int]TransferWorldview, 
	) {
	updatedWorldviewToAssignerCh <- copyWorldviews(latestWorldviews)
	updatedWorldviewToNetworkCh <- copyWorldviewsStringKey(latestWorldviews, myID)  
	updatedWorldviewToSyncCh <- copyWorldviews(latestWorldviews)
}

// _____________________________________________________________________________
// ----------FUNKSJONER FOR Å TA IMOT OG HÅNDTERE DATA FRA ANDRE MODULER--------
// _____________________________________________________________________________

// SYNC sin func
func updateWorldviewFromSync(latestWorldviews map[int]Worldview, inputSyncedHallOrders HallOrders, myID int) map[int]Worldview {
	worldviewsMap := latestWorldviews
	worldviewsMap[myID].hallOrders = inputSyncedHallOrders
	return worldviewsMap
}

// Får inn worldview fra network, bruker IDen til å legge til/oppdatere map
func updatePeerWorldviewFromNetwork(latestWorldviews map[int]Worldview, inputPeerWorldview Worldview) { // Ingrid
	worldviewsMap := latestWorldviews
	peerID := inputPeerWorldview.idElevator
	worldviewsMap[peerID] = inputPeerWorldview
	return worldviewsMap
}

// TODO
// funksjon som legger inn caborders/hallorders inn i din egen worldview. evt samle de sånn at vi kan bruke samme funksjon for de

// Setter state fra confirmet til uncondiremd og ownerID til peerDied, kjøres når heis dør
func markPeerDeadInHallOrders(hallOrders Hallorders, lostId int) Hallorder {
	ho := hallOrders

	for i, row := range ho{
		for j := range row {
			order := ho[i][j]	

	
			if (order.ownerID == lostID) && (order.orderSyncState == Confirmed) {
				order.orderSyncState = Unconfirmed
				order.ownerID = peerDied
				
				}	
			ho[i][j] = order
			}
		}
	return ho	
}

// Mottar elevatorState på channel fra FSM, bruke dette til å oppdatere state og
//  ordre i worldview.
func updateWorldviewWithElevatorState(worldview Worldview, inputStateElevator StateElevator) Worldview {
    wv := worldview
    wv.state = inputStateElevator
    floor := inputStateElevator.floor
    dir := inputStateElevator.dir

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

/*
Finne ut hvor disse funk skal stå. Sync?
-Sjekke om alle har en ordre som er på proposedDeleted -> Da skal den settes til No Order og No Owner og skru av lys via channel til FSM
-Må ha en funksjon som sjekker om alle har unconfirmed order -> Da skal den gjøre om til confirmed. Lys skal skru på via channel til FSM
*/

// Denne itererer gjennom alle hall orders og sjekker. Klarer ikke å tenke om det er nødvendig atm
func confirmIfAllAgree(worldviewsMap map[int]Worldview, myID int) (HallOrders, bool) {
	myOrders := worldviewsMap[myID].hallOrders
	changed := false

	// Itererer gjennom hall orders til vår heis
    for f := 0; f < NumFloors; f++ {
        for d := 0; d < Directions; d++ {
            order := myOrders[f][d]

			// Sjekker om vi har noen orders som er unconfirmed
            if order.syncState != Unconfirmed {
                continue
            }

			// Antar først at alle er enige. Hvis noen andre har ordersyncstate til None så settes den til false
			// Må den gjelde noen andre enn false?
            allAgree := true
            for _, peer := range worldviewsMap {
                peerState := peer.hallOrders[f][d].syncState
                if peerState == None {
                    allAgree = false
                    break
                }
            }

            // Hvis alle er enige, så oppdater staten 
            if allAgree {
                myOrders[f][d] = Order{
                    syncState: Confirmed,
					// Setter ownerID etter den har vært i assigned. Evt trenger vi den? Ja tror det. Hvor skal den være?
                    ownerID:   noOwner,
                }
            }
        }
    }
    return myOrders, changed
}

func deleteIfAllAgree(worldviewsMap map[int]Worldview, myID int) (HallOrders, bool) {
	myOrders := worldviewsMap[myID].hallOrders
	changed := false

    for f := 0; f < NumFloors; f++ {
        for d := 0; d < Directions; d++ {
            if myOrders[f][d].syncState != DeleteProposed {
                continue
            }

            allAgree := true
            for _, peer := range worldviewsMap {
                peerState := peer.hallOrders[f][d].syncState
                if peerState != DeleteProposed && peerState != None {
                    allAgree = false
                    break
                }
            }


            if allAgree {
                myOrders[f][d] = Order{
                    syncState: None,
                    ownerID:   noOwner,
                }
                changed = true
            }
        }
    }
    return myOrders, changed
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

	worldviewToAssignerCh   chan<- map[int]TransferWorldview,
	worldviewToSyncCh       chan<- map[int]Worldview,
	worldviewToNetworkCh    chan<- map[string]worldview,
	) { 
// NB!!!
// Pass på at channelsene bare sender inn når det skjer en endring, slik at de ikke blokkerer
	
	worldviewsMap := make(map[int]Worldview)
	myWorldview := worldviewsMap[myID]

	for {
		select {
		
		// Får inn endring i stateElevator fra FSM. Oppdaterer worldview med ny state og oppdaterer fullførte ordre
		case inputStateElevator := <-elevatorToWorldviewCh:
			myWorldview = updateWorldviewWithElevatorState(ourWorldview, inputStateElevator) // ingrid hjelp
			worldviewsMap[myID] = myWorldview
			sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
		
		
		
	// Får inn syncet hallorders fra sync. Den må da oppdatere worldview også sende den oppdaterte til andre moduler
		case inputSyncedHallOrders := <-syncToWorldviewCh:
			worldviewsMap = updateWorldviewFromSync(worldviewsMap, inputHallOrders, myID)
			sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
		
		// Får inn en peers worldview. Må Oppdatere map og sende til andre moduler
		case inputPeerWorldview := <-networkToWorldviewCh:
			worldviewsMap = updatePeerWorldviewFromNetwork(worldviewsMap, inputPeerWorldview)
			sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
		
		// Får inn at en peer lever
		// TODO: Sette disse to sammen og bruke samme channel
		case inputNewPeer := <-newPeerIdCh:
			worldviewsMap = updatePeerWorldviewFromNetwork(worldviewsMap, inputNewPeer)
			sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
		
		// Får inn at en peer er død
		case inputDeadPeer := <-lostPeerIdCh:
			// TODO: Hvordan sette node død?
			// peerdead funksjonen må kjøres her et sted. 
			worldviewsMap = HandleLostPeer(worldviewsMap, myID, inputDeadPeer)
			sendWorldviewsToOtherModules(worldviewsMap, worldviewToNetworkCh, worldviewToAssignerCh, worldviewToSyncCh, myID)
		
		case 
			}
		} 
	}





















//___________________________________________________________________________
//------FUNKSJONER FOR Å SENDE KOPIERT WORLDVIEW TIL ANDRE MODULER-----------
//___________________________________________________________________________

type HallOrdersPublic [NumFloors][Directions]Order

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

// Tar inn map, setter den døde noden sin state til død og oppdaterer ordre til død node
func HandleLostPeer(latestWorldviews map[int]Worldview, myID int, lostID int) map[int]Worldview{
	lwv := latestWorldviews
	///lwv[lostID].state = dead  ???
    wv := lwv[myID]

    wv.HallOrders = markPeerDeadInHallOrders(wv.HallOrders, lostID)
 
    lwv[myID] = wv

	return lwv
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

