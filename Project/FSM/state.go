package fsm

import (
	elevio "Project/Driver"
	t "Project/types"
)

const N_FLOORS = t.N_FLOORS
const N_BUTTONS = t.N_BUTTONS

type Button = t.Button
type Behaviour = t.Behaviour
type Direction = t.Direction
type ElevatorState = t.ElevatorState

const B_HallUp = t.B_HallUp
const B_HallDown = t.B_HallDown
const B_Cab = t.B_Cab

const EB_Idle = t.EB_Idle
const EB_DoorOpen = t.EB_DoorOpen
const EB_Moving = t.EB_Moving

const D_Down = t.D_Down
const D_Stop = t.D_Stop
const D_Up = t.D_Up

func InitElevatorState() ElevatorState {
	return t.InitElevatorState()
}

// sendState sender alltid siste elevatorState til worldview, og dropper eventuelle gamle verdier i kanalen.
func sendState(elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	select {
	case elevatorStateCh <- *elevatorState:
	default:
		select {
		case <-elevatorStateCh:
		default:
		}
		elevatorStateCh <- *elevatorState
	}
}

func updateFloor(floor int, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	elevio.SetFloorIndicator(floor)
	if elevatorState.Floor == floor {
		return
	}

	if (floor == 0 && elevatorState.Dirn == D_Down) || (floor == N_FLOORS-1 && elevatorState.Dirn == D_Up) {
		elevio.SetMotorDirection(elevio.MD_Stop)
		elevatorState.Dirn = D_Stop
	}

	elevatorState.Floor = floor
	sendState(elevatorState, elevatorStateCh)
}

func updateErrorState(errorState bool, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) bool {
	if elevatorState.Error == errorState {
		return elevatorState.Error
	}
	elevatorState.Error = errorState
	sendState(elevatorState, elevatorStateCh)
	return elevatorState.Error
}
