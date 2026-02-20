/*package main

import (
	"fmt"
)

func main() {
	e := elevator.Elevator{
		floor: 1,
		dirn:  elevator.D_Up,
		requests: [4][3]int{
			{0, 0, 0}, // floor 0
			{1, 0, 1}, // floor 1: HallUp + Cab
			{0, 1, 0}, // floor 2: HallDown
			{0, 0, 0}, // floor 3
		},
	}

	elevator.elevator_print(e)

	pair := elevator.requests_chooseDirection(e)
	fmt.Printf("chooseDirection: dir=%s, behaviour=%s\n",
		elevator.elevator_dirnToString(pair.dirn),
		elevator.elevator_behaviorToString(pair.behaviour))

	fmt.Printf("shouldStop: %v\n", elevator.requests_shouldStop(e))
}*/

package main

import "fmt"

// ===== ALL YOUR ELEVATOR CODE HERE =====

// Constants
const N_FLOORS = 4
const N_BUTTONS = 3

// Types
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
	D_Up Direction = iota
	D_Down
	D_Stop
)

type DirnBehaviourPair struct {
	dirn      Direction
	behaviour Behaviour
}

type Elevator struct {
	floor     int
	dirn      Direction
	behaviour Behaviour
	requests  [N_FLOORS][N_BUTTONS]int
	config    struct {
		doorOpenDuration_s float64
	}
}

// Elevator functions (from elevator.c)
func elevator_behaviorToString(eb Behaviour) string {
	switch eb {
	case EB_Idle:
		return "EB_Idle"
	case EB_DoorOpen:
		return "EB_DoorOpen"
	case EB_Moving:
		return "EB_Moving"
	default:
		return "EB_UNDEFINED"
	}
}

func elevator_dirnToString(d Direction) string {
	switch d {
	case D_Up:
		return "D_Up"
	case D_Down:
		return "D_Down"
	case D_Stop:
		return "D_Stop"
	default:
		return "D_UNDEFINED"
	}
}

func elevator_buttonToString(b Button) string {
	switch b {
	case B_HallUp:
		return "B_HallUp"
	case B_HallDown:
		return "B_HallDown"
	case B_Cab:
		return "B_Cab"
	default:
		return "B_UNDEFINED"
	}
}

func elevator_print(es Elevator) {
	fmt.Printf("  +--------------------+\n")
	fmt.Printf("  |floor = %-2d          |\n", es.floor)
	fmt.Printf("  |dirn  = %-12.12s|\n", elevator_dirnToString(es.dirn))
	fmt.Printf("  |behav = %-12.12s|\n", elevator_behaviorToString(es.behaviour))
	fmt.Printf("  +--------------------+\n")
	fmt.Printf("  |  | up  | dn  | cab |\n")
	for f := N_FLOORS - 1; f >= 0; f-- {
		fmt.Printf("  | %d", f)
		for btn := 0; btn < N_BUTTONS; btn++ {
			if (f == N_FLOORS-1 && btn == int(B_HallUp)) || (f == 0 && btn == int(B_HallDown)) {
				fmt.Printf("|     ")
			} else {
				if es.requests[f][btn] != 0 {
					fmt.Printf("|  #  ")
				} else {
					fmt.Printf("|  -  ")
				}
			}
		}
		fmt.Printf("|\n")
	}
	fmt.Printf("  +--------------------+\n")
}

// Requests functions (from requests.c)
func requests_above(e Elevator) bool {
	for f := e.floor + 1; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.requests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

func requests_below(e Elevator) bool {
	for f := 0; f < e.floor; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.requests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

func requests_here(e Elevator) bool {
	for btn := 0; btn < N_BUTTONS; btn++ {
		if e.requests[e.floor][btn] != 0 {
			return true
		}
	}
	return false
}

func requests_chooseDirection(e Elevator) DirnBehaviourPair {
	switch e.dirn {
	case D_Up:
		if requests_above(e) {
			return DirnBehaviourPair{D_Up, EB_Moving}
		}
		if requests_here(e) {
			return DirnBehaviourPair{D_Down, EB_DoorOpen}
		}
		if requests_below(e) {
			return DirnBehaviourPair{D_Down, EB_Moving}
		}
		return DirnBehaviourPair{D_Stop, EB_Idle}
	case D_Down:
		if requests_below(e) {
			return DirnBehaviourPair{D_Down, EB_Moving}
		}
		if requests_here(e) {
			return DirnBehaviourPair{D_Up, EB_DoorOpen}
		}
		if requests_above(e) {
			return DirnBehaviourPair{D_Up, EB_Moving}
		}
		return DirnBehaviourPair{D_Stop, EB_Idle}
	case D_Stop:
		if requests_here(e) {
			return DirnBehaviourPair{D_Stop, EB_DoorOpen}
		}
		if requests_above(e) {
			return DirnBehaviourPair{D_Up, EB_Moving}
		}
		if requests_below(e) {
			return DirnBehaviourPair{D_Down, EB_Moving}
		}
		return DirnBehaviourPair{D_Stop, EB_Idle}
	default:
		return DirnBehaviourPair{D_Stop, EB_Idle}
	}
}

func requests_shouldStop(e Elevator) bool {
	switch e.dirn {
	case D_Down:
		return e.requests[e.floor][B_HallDown] != 0 ||
			e.requests[e.floor][B_Cab] != 0 ||
			!requests_below(e)
	case D_Up:
		return e.requests[e.floor][B_HallUp] != 0 ||
			e.requests[e.floor][B_Cab] != 0 ||
			!requests_above(e)
	default:
		return true
	}
}

// ===== YOUR TEST CODE =====
func main() {
	e := Elevator{
		floor: 1,
		dirn:  D_Up,
		requests: [4][3]int{
			{0, 0, 0}, // floor 0
			{1, 0, 1}, // floor 1: HallUp + Cab
			{0, 1, 0}, // floor 2: HallDown
			{0, 0, 0}, // floor 3
		},
	}

	elevator_print(e)

	pair := requests_chooseDirection(e)
	fmt.Printf("chooseDirection: dir=%s, behaviour=%s\n",
		elevator_dirnToString(pair.dirn),
		elevator_behaviorToString(pair.behaviour))

	fmt.Printf("shouldStop: %v\n", requests_shouldStop(e))
}
