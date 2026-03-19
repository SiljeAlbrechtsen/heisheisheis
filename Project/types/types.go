package types


const N_FLOORS = 4
const N_BUTTONS = 3
const N_DIRECTIONS = 2

type Button int

const (
	B_HallUp Button = iota
	B_HallDown
	B_Cab
)

type Behaviour int

const (
	EB_Idle Behaviour = iota
	EB_DoorOpen
	EB_Moving
)

type Direction int

const (
	D_Down Direction = -1
	D_Stop Direction = 0
	D_Up   Direction = 1
)

type ElevatorState struct {
	Floor     int
	Dirn      Direction
	Behaviour Behaviour
	Requests  [N_FLOORS][N_BUTTONS]bool
	Error     bool
}

type OrderSyncState int

const (
	None OrderSyncState = iota
	Unconfirmed
	Confirmed
	DeleteProposed
)

type Order struct {
	SyncState OrderSyncState
	OwnerID   string
}

type HallOrders [N_FLOORS][2]Order
type AssignmentMatrix [N_FLOORS][N_BUTTONS]bool

// Worldview type — used across FSM, worldview, and assignment packages
type Worldview struct {
	IdElevator   string
	HallOrders   HallOrders
	State        ElevatorState
	AllCabOrders map[string][N_FLOORS]bool
	ErrorState   bool // Settes ved motorstopp/obstruction
	Dead         bool // Settes ved nettverkstap
}

