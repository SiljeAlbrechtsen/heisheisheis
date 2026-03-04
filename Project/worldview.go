package assignment

type OrderState uint8
type ElevID string
const Directions = 2
//type CabOrders [NumFloors]bool // Må vel ikke deklareres først?


const (
    None OrderState = iota
    Unconfirmed
    Confirmed
    DeleteProposed
)

type HallOrders [NumFloors][Directions]OrderState

type StateElevator = Elevator

// Struct for egen worldview
type WorldView struct {
	IDelevator uint8
	StateElevator // TODO: Må hente type fra fsm
	hallOrders HallOrders
	mycabOrders [NumFloors]bool // En liste med true or false for hver eneste etasje å trykke inn
}

// Struct der alle sine worldviews 
type MergedWorldviews struct {
	Elevators map[ElevID]ElevState
}


func EqualHallOrders(worldviews map[string]WorldView) bool {
	if (len(worldviews) <= 1){
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
func nextOrderState(current OrderState) OrderState {
	switch current {
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

