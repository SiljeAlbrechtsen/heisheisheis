package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"Driver-go/elevio"
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

func elevator_uninitialized() Elevator {
	addr := resolveElevatorAddr()
	elevio.Init(addr, N_FLOORS)
	return Elevator{
		floor:     -1,
		dirn:      D_Stop,
		behaviour: EB_Idle,
		config: struct {
			doorOpenDuration_s float64
		}{doorOpenDuration_s: 3.0},
	}
}

func resolveElevatorAddr() string {
	if addr := strings.TrimSpace(os.Getenv("ELEVATOR_ADDR")); addr != "" {
		return addr
	}
	candidates := []string{"localhost:15657"}
	if out, err := exec.Command("sh", "-c", "ip route | awk '/default/ {print $3}'").Output(); err == nil {
		ip := strings.TrimSpace(string(out))
		if ip != "" {
			candidates = append(candidates, ip+":15657")
		}
	}
	for _, addr := range candidates {
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return addr
		}
	}
	return candidates[0]
}

func elevator_floorSensor() int {
	return elevio.GetFloor()
}

func elevator_stopButton() int {
	if elevio.GetStop() {
		return 1
	}
	return 0
}

func elevator_obstruction() int {
	if elevio.GetObstruction() {
		return 1
	}
	return 0
}

func elevator_requestButton(floor int, button Button) int {
	if elevio.GetButton(elevio.ButtonType(button), floor) {
		return 1
	}
	return 0
}

func elevator_requestButtonLight(floor, btn int, on int) {
	elevio.SetButtonLamp(elevio.ButtonType(btn), floor, on != 0)
}

func elevator_motorDirection(d Direction) {
	elevio.SetMotorDirection(elevio.MotorDirection(d))
}

func elevator_doorLight(on int) {
	elevio.SetDoorOpenLamp(on != 0)
}

func elevator_floorIndicator(floor int) {
	elevio.SetFloorIndicator(floor)
}

func elevator_stopButtonLight(on int) {
	elevio.SetStopLamp(on != 0)
}

func elevator_buttonToString(b Button) string {
	return fmt.Sprintf("BTN_%d", b)
}

func elevator_print(e Elevator) {
	fmt.Printf("floor=%d dir=%s beh=%s\n", e.floor, elevator_dirnToString(e.dirn), elevator_behaviorToString(e.behaviour))
}
func elevator_dirnToString(d Direction) string {
	switch d {
	case D_Down:
		return "Down"
	case D_Up:
		return "Up"
	default:
		return "Stop"
	}
}
func elevator_behaviorToString(b Behaviour) string {
	switch b {
	case EB_DoorOpen:
		return "DoorOpen"
	case EB_Moving:
		return "Moving"
	default:
		return "Idle"
	}
}

