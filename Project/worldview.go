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

func nextOrderState(currentOrder Order) orderSyncState {
	switch currentOrder.OrderSyncState {
	case None:
		return Unconfirmed
	case Unconfirmed:
		return Confirmed
	case Confirmed:
		return DeleteProposed
	case DeleteProposed:
		return None
	default:
		return None
	}
}

// Trigges når vi får inn nye worldviews
func syncHallOrders(latestWorldviews map[int]Worldview) HallOrders {
	myHallOrders := latestWorldviews[0].hallOrders // TODO: MAGIC NUMBER

	// Itererer gjennom hele map. TODO: itererer også gjennom seg selv
	for _, w := range latestWorldviews {
		//Iterere gjennom hallOrdersene
		for f := range NumFloors {
			for d := range Directions {
				//
				myCurrentOrder := myHallOrders[f][d]
				peerCurrentOrder := w.hallOrders[f][d]
				if nextOrderState(myCurrentOrder) == peerCurrentOrder.orderSyncState {
					if myCurrentOrder.ownerID == peerDied {
						w.hallOrders[f][d] = myCurrentOrder
					} else {
						myHallOrders[f][d] = peerCurrentOrder

					}
				}
			}
		}
	}

	// latestWorldviews[0].hallOrders = myHallOrders
	return myHallOrders // TODO Må ha med linjen over et annet sted
}

/*
if heisA.nextorder == heisB.stateorder
	if heisA.ownerID.peerDied
		blir heisB.stateorder == heisA.state
	else
		blir heisA.stateorder == heisB.stateOrder
Funk skal:
Sammenligne SyncOrder
- HVis de er like return
- Hvis ikke:
	- Må sjekke ownerID + sync
	- Først sjekke state så ID pga

	- Hvis i none + unconfirmed = u

Må loope gjennom alle aktive heiser sine hall orders
Det må være dobbel løkke siden det er en matrise

Vi må sette våres heis til første, så skal alle andre sine hallorders sammenlignes med denne

for gjennom hele map


Det holder hvis alle skal frem i tid
Tilfellet det ikke er riktig er hvis owner ID = peerDied. Da er det motsatt
*/

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
	*/

}

func addMergedHallOrdersToWorldview(worldview map[string]WorldView, mergedHallOrders HallOrders) map[string]WorldView {

}

// TODO
// funksjon for å sette state fra confirmet til uncondiremd og ownerID til peerDied når heis dør
