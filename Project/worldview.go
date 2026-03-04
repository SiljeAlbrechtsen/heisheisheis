package assignment

type OrderState uint8
type ElevID string
type Directions uint8 = 2
//type CabOrders [NumFloors]bool // Må vel ikke deklareres først?


const (
    None OrderState = iota
    Unconfirmed
    Confirmed
    DeleteProposed
	noOrders
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

lastWorldview := make(map[string]Worldview)


func hallOrdersEqual(worldview map[string]WorldView) bool {
	if len(worldview) <= 1{
		return true	
	}

	var reference [NumFloors][NumButtons]OrderState
	first := true

	for _, w := range worldview { // for key, value TODO: Bytte w med noe annet
		if first {
			reference = w.hallOrders
			first = false
		}
		if w.HallOrders != reference { // Go kan sammenligne arrays direkte
			return false
		}
	}
	return true		
}

func MergeWorldviews