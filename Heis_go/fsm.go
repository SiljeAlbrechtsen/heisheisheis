package main

import (
	"fmt"
)

// You must define these types and functions somewhere else, mirroring your C code:
//
// type Elevator struct {
//     floor      int
//     dirn       Direction
//     behaviour  Behaviour
//     requests   [N_FLOORS][N_BUTTONS]int
//     config     struct {
//         doorOpenDuration_s float64
//     }
// }
//
// type Button int
// type Behaviour int
// type Direction int
//
// type DirnBehaviourPair struct {
//     dirn      Direction
//     behaviour Behaviour
// }
//
// const (
//     EB_DoorOpen Behaviour = iota
//     EB_Moving
//     EB_Idle
// )
//
// const (
//     D_Down Direction = iota
//     D_Stop
//     // ...
// )
//
// const N_FLOORS = 4
// const N_BUTTONS = 3
//
// func elevator_requestButtonLight(floor, btn int, on int) {}
// func elevator_motorDirection(d Direction)               {}
// func elevator_buttonToString(b Button) string           { return "" }
// func elevator_print(e Elevator)                         {}
// func elevator_doorLight(on int)                         {}
// func elevator_floorIndicator(floor int)                 {}
//
// func requests_shouldClearImmediately(e Elevator, floor int, btn Button) bool { return false }
// func requests_shouldStop(e Elevator) bool                                    { return false }
// func requests_clearAtCurrentFloor(e Elevator) Elevator                       { return e }
// func requests_chooseDirection(e Elevator) DirnBehaviourPair                  { return DirnBehaviourPair{} }
//
// func TimerStart(duration float64) {}
//
// (TimerStart is the Go/`timer_start` from your previous translation.)

// setAllLights is the Go equivalent of static void setAllLights(Elevator es)
func setAllLights(es Elevator) {
	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			elevator_requestButtonLight(floor, btn, es.requests[floor][btn])
		}
	}
}

// fsm_onInitBetweenFloors(Elevator* e)
func FsmOnInitBetweenFloors(e *Elevator) {
	elevator_motorDirection(D_Down)
	e.dirn = D_Down
	e.behaviour = EB_Moving
}

// fsm_onRequestButtonPress(Elevator* e, int btn_floor, Button btn_type)
func FsmOnRequestButtonPress(e *Elevator, btnFloor int, btnType Button) {
	fmt.Printf("\n\n%s(%d, %s)\n", "FsmOnRequestButtonPress", btnFloor, elevator_buttonToString(btnType))
	elevator_print(*e)

	switch e.behaviour {
	case EB_DoorOpen:
		if requests_shouldClearImmediately(*e, btnFloor, btnType) {
			TimerStart(e.config.doorOpenDuration_s)
		} else {
			e.requests[btnFloor][btnType] = 1
		}

	case EB_Moving:
		e.requests[btnFloor][btnType] = 1

	case EB_Idle:
		e.requests[btnFloor][btnType] = 1
		pair := requests_chooseDirection(*e)
		e.dirn = pair.dirn
		e.behaviour = pair.behaviour

		switch pair.behaviour {
		case EB_DoorOpen:
			elevator_doorLight(1)
			TimerStart(e.config.doorOpenDuration_s)
			*e = requests_clearAtCurrentFloor(*e)

		case EB_Moving:
			elevator_motorDirection(e.dirn)

		case EB_Idle:
			// nothing
		}
	}

	setAllLights(*e)

	fmt.Printf("\nNew state:\n")
	elevator_print(*e)
}

// fsm_onFloorArrival(Elevator* e, int newFloor)
func FsmOnFloorArrival(e *Elevator, newFloor int) {
	fmt.Printf("\n\n%s(%d)\n", "FsmOnFloorArrival", newFloor)
	elevator_print(*e)

	e.floor = newFloor
	elevator_floorIndicator(e.floor)

	switch e.behaviour {
	case EB_Moving:
		if requests_shouldStop(*e) {
			elevator_motorDirection(D_Stop)
			elevator_doorLight(1)
			*e = requests_clearAtCurrentFloor(*e)
			TimerStart(e.config.doorOpenDuration_s)
			setAllLights(*e)
			e.behaviour = EB_DoorOpen
		}
	default:
		// nothing
	}

	fmt.Printf("\nNew state:\n")
	elevator_print(*e)
}

// fsm_onDoorTimeout(Elevator* e)
func FsmOnDoorTimeout(e *Elevator) {
	fmt.Printf("\n\n%s()\n", "FsmOnDoorTimeout")
	elevator_print(*e)

	switch e.behaviour {
	case EB_DoorOpen:
		pair := requests_chooseDirection(*e)
		e.dirn = pair.dirn
		e.behaviour = pair.behaviour

		switch e.behaviour {
		case EB_DoorOpen:
			TimerStart(e.config.doorOpenDuration_s)
			*e = requests_clearAtCurrentFloor(*e)
			setAllLights(*e)

		case EB_Moving, EB_Idle:
			elevator_doorLight(0)
			elevator_motorDirection(e.dirn)
		}
	default:
		// nothing
	}

	fmt.Printf("\nNew state:\n")
	elevator_print(*e)
}
