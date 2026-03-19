package fsm

import (
	elevio "Project/Driver"
	elev "Project/elevator"
)

type Button = elev.Button
type Behaviour = elev.Behaviour
type Direction = elev.Direction
type ElevatorState = elev.ElevatorState

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

	if (floor == 0 && elevatorState.Dirn == elev.D_Down) || (floor == elev.N_FLOORS-1 && elevatorState.Dirn == elev.D_Up) {
		elevio.SetMotorDirection(elevio.MD_Stop)
		elevatorState.Dirn = elev.D_Stop
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
