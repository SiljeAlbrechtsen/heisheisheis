package elevator

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

func InitElevatorState() ElevatorState {
	return ElevatorState{
		Floor:     -1,
		Dirn:      D_Stop,
		Behaviour: EB_Idle,
	}
}
