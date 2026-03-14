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

func InitElevator(elevator *ElevatorState) {
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
	elevator.Floor = elevio.GetFloor()
}

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
