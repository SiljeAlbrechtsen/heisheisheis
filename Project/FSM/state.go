package fsm

import (
	elevio "Project/Driver"
)

const N_FLOORS = 4
const N_BUTTONS = 3

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
	config    struct {
		doorOpenDuration_s float64
	}
}

func InitElevatorState() ElevatorState { //lager en state
	addr := resolveElevatorAddr()
	elevio.Init(addr, N_FLOORS)
	return ElevatorState{
		Floor:     -1,
		Dirn:      D_Stop,
		Behaviour: EB_Idle,
		config: struct {
			doorOpenDuration_s float64
		}{doorOpenDuration_s: 3.0},
	}
}

//////////////Opdater state og send til worldview/////////////////////

func UpdateFloor(floor int, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	if elevatorState.Floor == floor {
		return
	}
	elevatorState.Floor = floor
	elevatorStateCh <- *elevatorState
}

func UpdateDirection(direction Direction, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	if elevatorState.Dirn == direction {
		return
	}
	elevio.SetMotorDirection(elevio.MotorDirection(direction))
	elevatorState.dirn = direction
	elevatorStateCh <- *elevatorState
}

func UpdateBehaviour(behaviour Behaviour, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	if elevatorState.Behaviour == behaviour {
		return
	}
	elevatorState.Behaviour = behaviour
	elevatorStateCh <- *elevatorState
}

func UpdateRequests(requests [N_FLOORS][N_BUTTONS]bool, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	if elevatorState.Requests == requests {
		return
	}
	elevatorState.Requests = requests
	elevatorStateCh <- *elevatorState
}
