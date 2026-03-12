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

func FSM2(assignerToFsmCh chan [N_FLOORS][N_BUTTONS]bool, elevatorStateCh chan ElevatorState) {

	elevatorState := InitElevatorState()

	InitElevator(&elevatorState)

	floorTicker := time.NewTicker(50 * time.Millisecond)
	defer floorTicker.Stop()

	fmt.Println("FSM started")
	var doorTimer <-chan time.Time

	for {
		select {
		case newRequests := <-assignerToFsmCh:
			// Merger inn nye requests: assigner kan bare sette true, aldri false.
			// Clearing skjer kun via requests_clearAtCurrentFloor når heisen betjener en etasje.
			merged := elevatorState.Requests
			for f := 0; f < N_FLOORS; f++ {
				for b := 0; b < N_BUTTONS; b++ {
					if newRequests[f][b] {
						merged[f][b] = true
					}
				}
			}
			if merged != elevatorState.Requests {
				UpdateRequests(merged, &elevatorState, elevatorStateCh)

				if elevatorState.Floor == -1 || doorTimer != nil {
					continue
				}

				db := requests_chooseDirection(elevatorState)
				if db.ElevatorBehaviour == EB_DoorOpen {
					elevio.SetMotorDirection(elevio.MD_Stop)
					elevio.SetDoorOpenLamp(true)
					elevatorState = requests_clearAtCurrentFloor(elevatorState)
					UpdateRequests(elevatorState.Requests, &elevatorState, elevatorStateCh)
					UpdateBehaviour(EB_DoorOpen, &elevatorState, elevatorStateCh)
					doorTimer = time.After(3000 * time.Millisecond)
				} else {
					if db.Dirn != elevatorState.Dirn {
						UpdateDirection(db.Dirn, &elevatorState, elevatorStateCh)
						elevio.SetMotorDirection(elevio.MotorDirection(db.Dirn))
					}
					UpdateBehaviour(db.ElevatorBehaviour, &elevatorState, elevatorStateCh)
				}
			}

		case <-doorTimer:
			elevio.SetDoorOpenLamp(false)
			doorTimer = nil

			db := requests_chooseDirection(elevatorState)
			UpdateDirection(db.Dirn, &elevatorState, elevatorStateCh)
			UpdateBehaviour(db.ElevatorBehaviour, &elevatorState, elevatorStateCh)
			elevio.SetMotorDirection(elevio.MotorDirection(db.Dirn))

		case <-floorTicker.C:
			sensorFloor := elevio.GetFloor()
			if sensorFloor != -1 {
				UpdateFloor(sensorFloor, &elevatorState, elevatorStateCh)
			}

			if doorTimer != nil || elevatorState.Floor == -1 || elevatorState.Behaviour != EB_Moving {
				continue
			}

			if requests_shouldStop(elevatorState) {
				elevio.SetMotorDirection(elevio.MD_Stop)
				elevio.SetDoorOpenLamp(true)
				elevatorState = requests_clearAtCurrentFloor(elevatorState)
				UpdateRequests(elevatorState.Requests, &elevatorState, elevatorStateCh)
				UpdateDirection(D_Stop, &elevatorState, elevatorStateCh)
				UpdateBehaviour(EB_DoorOpen, &elevatorState, elevatorStateCh)
				doorTimer = time.After(3000 * time.Millisecond)
				continue
			}

			db := requests_chooseDirection(elevatorState)
			if db.Dirn != elevatorState.Dirn {
				UpdateDirection(db.Dirn, &elevatorState, elevatorStateCh)
				elevio.SetMotorDirection(elevio.MotorDirection(db.Dirn))
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