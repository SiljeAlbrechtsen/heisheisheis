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


// Mottar elevatorState på channel fra FSM, bruke dette til å oppdatere worldview med data.