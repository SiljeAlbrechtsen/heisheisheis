package main // endre til FSM

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	elevio "../Driver"
)

///Kan fjernes kanskje? heller importere fra en mer definert pakke?
////////////////////////////////////////////////////////////////////

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
	floor     int
	dirn      Direction
	behaviour Behaviour
	requests  [N_FLOORS][N_BUTTONS]bool
	config    struct {
		doorOpenDuration_s float64
	}
}

func InitElevatorState() ElevatorState {
	addr := resolveElevatorAddr()
	elevio.Init(addr, N_FLOORS)
	return ElevatorState{
		floor:     -1,
		dirn:      D_Stop,
		behaviour: EB_Idle,
		config: struct {
			doorOpenDuration_s float64
		}{doorOpenDuration_s: 3.0},
	}
}

func resolveElevatorAddr() string { //Sjekke denne, skjønne hvordan den løser addr, og om den å evt fjernes/forenkles
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

/////////////////////////////////////////////////////////////////////////////////

func InitElevator(elevator *ElevatorState) { //Kjører til en etasje og stopper ved første etasje nedover, eller hvis den allerede er i en etasje så gjør den ingenting
	if elevio.GetFloor() == -1 {
		fmt.Println("Elevator is between floors, moving down to nearest floor")
		elevio.SetMotorDirection(elevio.MD_Down)
		for elevio.GetFloor() == -1 {
			time.Sleep(50 * time.Millisecond)
		}
		elevio.SetMotorDirection(elevio.MD_Stop)
		fmt.Printf("Arrived at floor %d\n", elevio.GetFloor())
	} else {
		fmt.Printf("Elevator is at floor %d\n", elevio.GetFloor())
	}
}

func ClearFloorRequest(elevator *ElevatorState) { //Clearer llisten fra bunn opp så etg 0 -> etg 1 -> etg 2 -> etg 3 ...
	for floor := 0; floor < len(elevator.requests); floor++ {
		for button := 0; button < len(elevator.requests[floor]); button++ {
			if elevator.requests[floor][button] {
				MoveToFloor(elevator, floor)
				elevator.requests[floor][button] = false
				PrintElevatorState(*elevator)
			}
		}
	}
}

func MoveToFloor(elevator *ElevatorState, targetFloor int) int {
	currentFloor := elevio.GetFloor()
	for {
		if elevio.GetFloor() != -1 {
			currentFloor = elevio.GetFloor()
			fmt.Println(currentFloor) // Slett etter testing
		}
		if targetFloor > currentFloor {
			elevio.SetMotorDirection(elevio.MD_Up)
		} else if targetFloor < currentFloor {
			elevio.SetMotorDirection(elevio.MD_Down)
		} else {
			elevio.SetMotorDirection(elevio.MD_Stop)
			fmt.Printf("Arrived at floor %d\n", targetFloor)
			ServeFloor(elevator)
			return elevio.GetFloor()
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func ServeFloor(elevator *ElevatorState) { //stopper åpner dør og venter i 3 sek før den lukker døren igjen
	fmt.Println("Serving floor")
	elevator.behaviour = EB_DoorOpen
	elevio.SetDoorOpenLamp(true)
	time.Sleep(3000 * time.Millisecond) //TO DO fjerne hard constant
	elevator.behaviour = EB_Idle
	elevio.SetDoorOpenLamp(false)
}

//////////////Test og hjelpe funksjoner kan slettes////////////////

func PrintElevatorState(e ElevatorState) {
	fmt.Printf("Floor: %d\nDirection: %d\nBehaviour: %d\nRequests: %v\n", e.floor, e.dirn, e.behaviour, e.requests)
}
