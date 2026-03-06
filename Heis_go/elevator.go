package elevator

import "fmt"


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
	EB_DoorOpen Behaviour = iota
	EB_Moving
	EB_Idle
)

type Direction int
const (
	D_Down Direction = iota
	D_Up
	D_Stop
)

type DirnBehaviourPair struct {
	dirn      Direction
	behaviour Behaviour
}

type Elevator struct {
	floor      int
	dirn       Direction
	behaviour  Behaviour
	requests   [N_FLOORS][N_BUTTONS]int
	config     struct {
		doorOpenDuration_s float64
	}
}

// elevator_* function stubs (replace with real implementations)
func elevator_requestButtonLight(floor, btn int, on int) {}
func elevator_motorDirection(d Direction)                {}
func elevator_buttonToString(b Button) string            { return fmt.Sprintf("BTN_%d", b) }
func elevator_print(e Elevator)                          {}
func elevator_doorLight(on int)                          {}
func elevator_floorIndicator(floor int)                  {}

// requests_* function stubs (implement in requests.go later)
func requests_shouldClearImmediately(e Elevator, floor int, btn Button) bool { return false }
func requests_shouldStop(e Elevator) bool                                    { return false }
func requests_clearAtCurrentFloor(e Elevator) Elevator                       { return e }
func requests_chooseDirection(e Elevator) DirnBehaviourPair                  { return DirnBehaviourPair{} }
