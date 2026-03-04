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
)

type HallOrders [NumFloors][Directions]OrderState

// Struct for egen worldview
type WorldView struct {
	IDelevator uint8
	StateElevator // TODO: få inn fra fsm
	HallOrders HallOrders
	CabOrders [NumFloors]bool // En liste med true or false for hver eneste etasje å trykke inn
}

// Struct der alle sine worldviews 
type MergedWorldviews struct {
	Elevators map[ElevID]ElevState

}