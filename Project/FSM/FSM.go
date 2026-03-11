package fsm

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	elevio "Project/Driver"
)

/*
Hver gang den endrer elevator state skal den sende oppdatering til worldview
- ankomst floor, obstruction, error +++
- lage GO routine for FSM
- At vi bare får endre elevator instansen en og en. Altså bare ha en funksjon for det.


Implementere knappetrykk i annen modul
- Skille på cab og hall orders
- Skal sende over til worldview
Done
*/

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

func InitElevator(elevator *ElevatorState) { //Kjører til en etasje og stopper ved første etasje nedover, eller hvis den allerede er i en etasje så gjør den ingenting
	if elevio.GetFloor() == -1 {
		fmt.Println("Elevator is between floors, moving down to nearest floor")
		elevio.SetMotorDirection(elevio.MD_Down)

		for elevio.GetFloor() == -1 { //pause fuksjonen til den kommer ned
			time.Sleep(50 * time.Millisecond)
		}

		elevio.SetMotorDirection(elevio.MD_Stop)

		fmt.Printf("Arrived at floor %d\n", elevio.GetFloor())
	} else {
		fmt.Printf("Elevator is at floor %d\n", elevio.GetFloor())
	}
	elevator.Floor = elevio.GetFloor() // endret
}

/////////////////////////////////////////////////////////////////////////////////

func FSM(requests chan [N_FLOORS][N_BUTTONS]bool, elevatorStateCh chan ElevatorState) { //sjekke om man kan endre [N_FLOORS][N_BUTTONS]bool til noe enklere navn

	elevatorState := InitElevatorState()

	InitElevator(&elevatorState)

	for {
		select {
		case newRequests := <-requests:
			if newRequests != elevatorState.Requests {
				elevatorState.Requests = newRequests
				//PrintElevatorState(elevatorState) //TO DO fjerne print
				MoveToFloor(&elevatorState, FindFloorFromRequest(elevatorState.Requests))
			}
		}
	}
}

func MoveToFloor(elevator *ElevatorState, targetFloor int) int { // TO DO endre til å bruke state update funksjonene
	currentFloor := elevio.GetFloor()
	for {
		if elevio.GetFloor() != -1 {
			currentFloor = elevio.GetFloor()
		}
		if targetFloor > currentFloor {
			elevio.SetMotorDirection(elevio.MD_Up)

		} else if targetFloor < currentFloor {
			elevio.SetMotorDirection(elevio.MD_Down)

		} else {
			elevio.SetMotorDirection(elevio.MD_Stop)
			fmt.Printf("Arrived at floor %d\n", targetFloor) //
			ServeFloor(elevator)
			return elevio.GetFloor()
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func ServeFloor(elevator *ElevatorState) { //stopper åpner dør og venter i 3 sek før den lukker døren igjen
	fmt.Println("Serving floor")
	elevator.Behaviour = EB_DoorOpen
	elevio.SetDoorOpenLamp(true)
	time.Sleep(3000 * time.Millisecond) //TO DO fjerne hard constant
	elevator.Behaviour = EB_Idle
	elevio.SetDoorOpenLamp(false)
}

///////////////////FSM2////////////////////
//Nye versjon, om denne funker kan alt over slettes

func FSM2(requests chan [N_FLOORS][N_BUTTONS]bool, elevatorStateCh chan ElevatorState) {

	elevatorState := InitElevatorState()

	InitElevator(&elevatorState)

	floorTicker := time.NewTicker(50 * time.Millisecond)
	defer floorTicker.Stop()

	targetFloor := -1
	fmt.Println("FSM started")
	var doorTimer <-chan time.Time // endret

	for {
		select {
		case newRequests := <-requests:
			if newRequests != elevatorState.Requests {
				UpdateRequests(newRequests, &elevatorState, elevatorStateCh)
				fmt.Println("Loop")
				fmt.Println(newRequests)

				targetFloor = FindFloorFromRequest(elevatorState.Requests) // endret

				if targetFloor == -1 {
					UpdateDirection(D_Stop, &elevatorState, elevatorStateCh)
					elevio.SetMotorDirection(elevio.MD_Stop) // endret
					UpdateBehaviour(EB_Idle, &elevatorState, elevatorStateCh)
					continue
				}

				if elevatorState.Floor != -1 && doorTimer == nil { // endret
					dir := MoveToFloor2(elevatorState.Floor, targetFloor)
					if dir != elevatorState.Dirn {
						UpdateDirection(dir, &elevatorState, elevatorStateCh)
						elevio.SetMotorDirection(elevio.MotorDirection(dir)) // endret
					}
					if dir != D_Stop {
						UpdateBehaviour(EB_Moving, &elevatorState, elevatorStateCh)
					}
				}
			}

		case <-doorTimer: // endret
			elevio.SetDoorOpenLamp(false) // endret
			UpdateBehaviour(EB_Idle, &elevatorState, elevatorStateCh) // endret
			doorTimer = nil // endret
			targetFloor = FindFloorFromRequest(elevatorState.Requests) // endret

		case <-floorTicker.C:
			sensorFloor := elevio.GetFloor()
			if sensorFloor != -1 {
				UpdateFloor(sensorFloor, &elevatorState, elevatorStateCh)
			}

			if doorTimer != nil { // endret
				continue // endret
			} // endret

			if targetFloor == -1 {
				targetFloor = FindFloorFromRequest(elevatorState.Requests)
				if targetFloor == -1 {
					continue
				}
			}

			if elevatorState.Floor == -1 {
				continue
			}

			if elevatorState.Floor == targetFloor {
				UpdateDirection(D_Stop, &elevatorState, elevatorStateCh)
				elevio.SetMotorDirection(elevio.MD_Stop) // endret
				UpdateBehaviour(EB_DoorOpen, &elevatorState, elevatorStateCh)
				elevio.SetDoorOpenLamp(true)
				doorTimer = time.After(3000 * time.Millisecond) // endret

				cleared := elevatorState.Requests
				for button := 0; button < N_BUTTONS; button++ {
					cleared[targetFloor][button] = false
				}
				UpdateRequests(cleared, &elevatorState, elevatorStateCh)
				targetFloor = -1 // endret
				continue
			}

			dir := MoveToFloor2(elevatorState.Floor, targetFloor)
			if dir != elevatorState.Dirn {
				UpdateDirection(dir, &elevatorState, elevatorStateCh)
				elevio.SetMotorDirection(elevio.MotorDirection(dir)) // endret
			}
			if dir != D_Stop {
				UpdateBehaviour(EB_Moving, &elevatorState, elevatorStateCh)
			}
		}
	}

}

func MoveToFloor2(currentFloor int, targetFloor int) Direction {
	if targetFloor > currentFloor {
		return D_Up
	} else if targetFloor < currentFloor {
		return D_Down
	}
	return D_Stop
}

//////////////Test og hjelpe funksjoner kan slettes////////////////

func PrintElevatorState(e ElevatorState) {
	fmt.Printf("Floor: %d\nDirection: %d\nBehaviour: %d\nRequests: %v\n", e.Floor, e.Dirn, e.Behaviour, e.Requests)
}

func UpdateRequest(requests [N_FLOORS][N_BUTTONS]bool, floor int, button elevio.ButtonType) [N_FLOORS][N_BUTTONS]bool {
	requests[floor][button] = true
	fmt.Printf("Updated---\n")
	return requests
}

func FindFloorFromRequest(request [N_FLOORS][N_BUTTONS]bool) int { //får en int fra request
	for floor := 0; floor < len(request); floor++ {
		for button := 0; button < len(request[floor]); button++ {
			if request[floor][button] {
				return floor
			}
		}
	}
	return -1 // endret
}