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
}

/////////////////////////////////////////////////////////////////////////////////

func FSM(requests chan [N_FLOORS][N_BUTTONS]bool, elevatorStateCh chan ElevatorState) { //sjekke om man kan endre [N_FLOORS][N_BUTTONS]bool til noe enklere navn

	elevatorState := InitElevatorState()

	InitElevator(&elevatorState)

	for {
		select {
		case newRequests := <-requests:
			if newRequests != elevatorState.requests {
				elevatorState.requests = newRequests
				//PrintElevatorState(elevatorState) //TO DO fjerne print
				MoveToFloor(&elevatorState, FindFloorFromRequest(elevatorState.requests))
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
	elevator.behaviour = EB_DoorOpen
	elevio.SetDoorOpenLamp(true)
	time.Sleep(3000 * time.Millisecond) //TO DO fjerne hard constant
	elevator.behaviour = EB_Idle
	elevio.SetDoorOpenLamp(false)
}

///////////////////FSM2////////////////////
//Nye versjon, om denne funker kan alt over slettes

func FSM2(requests chan [N_FLOORS][N_BUTTONS]bool, elevatorStateCh chan ElevatorState) { 

	elevatorState := InitElevatorState()

	InitElevator(&elevatorState)

	for {
		select {
		case newRequests := <-requests:
			if newRequests != elevatorState.requests {
				UpdateRequests(newRequests, &elevatorState, elevatorStateCh)

				targetFloor := FindFloorFromRequest(elevatorState.requests)
				for {
					sensorFloor := elevio.GetFloor()
					if sensorFloor != -1 {
						UpdateFloor(sensorFloor, &elevatorState, elevatorStateCh)
					}

					currentFloor := elevatorState.floor
					if currentFloor == -1 {
						time.Sleep(50 * time.Millisecond)
						continue
					}

					dir := MoveToFloor2(currentFloor, targetFloor)
					if dir != elevatorState.dirn {
						UpdateDirection(dir, &elevatorState, elevatorStateCh)
						elevio.SetMotorDirection(elevio.MotorDirection(dir))
					}

					if currentFloor == targetFloor {
						UpdateDirection(D_Stop, &elevatorState, elevatorStateCh)
						elevio.SetMotorDirection(elevio.MD_Stop)
						UpdateBehaviour(EB_DoorOpen, &elevatorState, elevatorStateCh)
						time.Sleep(3000 * time.Millisecond) //TODO fjerne hard constant
						UpdateBehaviour(EB_Idle, &elevatorState, elevatorStateCh)
						UpdateRequests([N_FLOORS][N_BUTTONS]bool{}, &elevatorState, elevatorStateCh)
						break
					}

					time.Sleep(100 * time.Millisecond)
				}
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
	fmt.Printf("Floor: %d\nDirection: %d\nBehaviour: %d\nRequests: %v\n", e.floor, e.dirn, e.behaviour, e.requests)
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
	return 0
}
