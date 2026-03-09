package worldview

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
	//
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
//type MergedWorldviews struct {
//	Elevators map[ElevID]ElevState
//}

//____________________________________________________________________________________________________________-
//----------------  FUNKSJONER FOR Å HÅNDTERE WORLDVIEW -------------------------------------------------------
//____________________________________________________________________________________________________________



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
        copied[id] = worldview.copyWorldview(worldview)
    }
    return copied
}


func sendWorldviewsToAssigner(latestWorldviews map[int]Worldview, updatedWorldviewToAssignerCh chan<- map[int]TransferWorldview)  {
	updatedWorldviewToAssignerCh<- copyWorldviews(latestWorldviews)
}

func sendWorldviewsToNetwork(latestWorldviews map[int]Worldview, updatedWorldviewToNetworkCh chan<- map[int]TransferWorldview)  {
	updatedWorldviewToNetworkCh <- copyWorldviews(latestWorldviews)
}

func sendHallRequestsToSync(worldview WorldView, )
/*
TODO:
Skrive GO-routines for assigner, worldview, sync
Finne ut hvordan håndtere data som kommer fra FSM
Finne ut hvordan håndtere data som kommer fra Network
Finne ut hvordan man kan ha network alene. Altså at den ikke er main
Finne ut hvordan vi trigger 
Finne ut hvordan vi gjør det når vi bare skal ha en worldview. Evt sette myID som en global const variabel som vi bruker som indeks.

*/

/*

func EqualHallOrders(worldviews map[string]WorldView) bool {
	if len(worldviews) <= 1 {
		return true
	}
	
	var reference [NumFloors][NumButtons]OrderState
	first := true
	
	for _, w := range worldviews { // for key, value TODO: Bytte w med noe annet
		if first {
			reference = w.hallOrders
			first = false
		}
		if w.hallOrders != reference { // Go kan sammenligne arrays direkte
			return false
		}
	}
	return true
}

func MergeHallOrders(worldview map[string]WorldView) HallOrders {
	// Case 1: Alle worldviews har like hallorders, returner disse
	
	// Case 2:
	/*
	Må iterere gjennom map.
	Må så iterere gjennom hver hall order
	case
	hvis
	
	?????????????
	
	
}

func addMergedHallOrdersToWorldview(worldview map[string]WorldView, mergedHallOrders HallOrders) map[string]WorldView {
	
}*/

// TODO
// funksjon for å sette state fra confirmet til uncondiremd og ownerID til peerDied når heis dør
