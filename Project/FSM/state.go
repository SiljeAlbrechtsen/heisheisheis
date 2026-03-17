package fsm

import (
	elevio "Project/Driver"
	t "Project/types"
	"fmt"
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

func InitElevatorState() ElevatorState { //A-TO DO: Sjekk om det trenges stor forbokstav
	return t.InitElevatorState()
}

//////////////Opdater state og send til worldview/////////////////////

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
	elevio.SetFloorIndicator(elevio.GetFloor())
	if elevatorState.Floor == floor {
		return
	}

	if floor == 0 && elevatorState.Dirn == D_Down || floor == N_FLOORS-1 && elevatorState.Dirn == D_Up {
		elevio.SetMotorDirection(elevio.MD_Stop)
		elevatorState.Dirn = D_Stop
	}

	elevatorState.Floor = floor
	sendState(elevatorState, elevatorStateCh)
}

// A-Tar i mot retning og oppdaterer state+channel til wv
func updateDirection(direction Direction, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	if elevatorState.Dirn == direction {
		return
	}
	elevatorState.Dirn = direction
	elevio.SetMotorDirection(elevio.MotorDirection(direction)) //A-TO DO: Sjekk om det trenges type konvertering og heller endre elevio sin type til å bruke vår Direction type
	sendState(elevatorState, elevatorStateCh)
}

func updateBehaviour(behaviour Behaviour, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	if elevatorState.Behaviour == behaviour {
		return
	}
	if behaviour == EB_DoorOpen {
		elevio.SetDoorOpenLamp(true)
	} else {
		elevio.SetDoorOpenLamp(false)
	}
	elevatorState.Behaviour = behaviour
	sendState(elevatorState, elevatorStateCh)
}

func updateRequests(requests [N_FLOORS][N_BUTTONS]bool, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	if elevatorState.Requests == requests {
		return
	}
	elevatorState.Requests = requests
	sendState(elevatorState, elevatorStateCh)
}

func updateBehaviourAndRequests(behaviour Behaviour, requests [N_FLOORS][N_BUTTONS]bool, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	changed := false

	if behaviour == EB_DoorOpen {
		elevio.SetDoorOpenLamp(true)
	} else {
		elevio.SetDoorOpenLamp(false)
	}
	if elevatorState.Behaviour != behaviour {
		elevatorState.Behaviour = behaviour
		changed = true
	}

	if elevatorState.Requests != requests {
		elevatorState.Requests = requests
		changed = true
	}

	if changed {
		sendState(elevatorState, elevatorStateCh)
		fmt.Printf("\n***\n%+v\n***\n", *elevatorState)
	}
}
