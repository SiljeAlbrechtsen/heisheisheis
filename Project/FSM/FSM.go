package fsm

//New version

import (
	"fmt"
	"time"

	elevio "Project/Driver"
	Hardware "Project/Hardware"
)

func InitElevator(elevator *ElevatorState, elevatorStateCh chan ElevatorState) {

	Hardware.TurnOffAllLights() //A-Fjerner alle gamle lys, i tilfelle heisen starter med noen lys på

	if elevio.GetFloor() == -1 {
		fmt.Println("*************\nElevator is between floors, moving down to nearest floor\n*************") //A-Fjern etter testing
		elevio.SetMotorDirection(elevio.MD_Down)

		for elevio.GetFloor() == -1 {
			time.Sleep(50 * time.Millisecond) //TO DO: HARD CONSTANT fjernt
		}

		elevio.SetMotorDirection(elevio.MD_Stop)
	}
	fmt.Println("*************\nElevator is at floor ", elevio.GetFloor(), "\n*************") //A-Fjern etter testing
	elevio.SetFloorIndicator(elevio.GetFloor())

	elevator.Floor = elevio.GetFloor()
	sendState(elevator, elevatorStateCh)
}

func updateElevatorRequests(elevatorState ElevatorState, newRequests [4][3]bool) ElevatorState {
	for f := 0; f < N_FLOORS; f++ {
		for b := 0; b < N_BUTTONS; b++ {
			if newRequests[f][b] {
				elevatorState.Requests[f][b] = true
			}
		}
	}
	return elevatorState
}

////////////////////////////////////////////////////////////////////////////

func FSM3(assignerToFsmCh chan [4][3]bool, elevatorStateCh chan ElevatorState) {

	elevatorState := InitElevatorState()
	InitElevator(&elevatorState, elevatorStateCh)

	floorTicker := time.NewTicker(50 * time.Millisecond) //A-TO DO: Fjern hardkoding
	defer floorTicker.Stop()

	var doorTimer <-chan time.Time

	stopBtnCh := make(chan bool)
	go elevio.PollStopButton(stopBtnCh)

	for { // 4 sjekk om det trenges å sjekke door open i should stop, 5 sjekk om det trenges å sjekke door open i should clear immediately, 6 Når den drar fra en etasje så stopper den i samme etasje og trur at den er i samme etg selvom den har forlatt etasjen
		select {
		case newRequests := <-assignerToFsmCh: //A-Tar i mot requests fra assigner og legger de i elevater sin state, sender så oppdatering til worldview
			//if requests_shouldClearImmediately() //
			mergedState := updateElevatorRequests(elevatorState, newRequests)
			updateRequests(mergedState.Requests, &elevatorState, elevatorStateCh)

			if doorTimer != nil && elevatorState.Floor != -1 && requests_shouldServeCurrentFloor(elevatorState) {
				elevatorState, doorTimer = openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
				continue
			}

			if doorTimer == nil {
				var served bool
				elevatorState, doorTimer, served = serveCurrentFloorNow(elevatorState, elevatorStateCh)
				if served {
					continue
				}
			}

			if elevatorState.Behaviour != EB_DoorOpen && requests_checkForRequests(elevatorState) {
				db := requests_chooseDirection(elevatorState)
				applyDecision(db, &elevatorState, elevatorStateCh)
			}

		case <-doorTimer:
			doorTimer = nil
			if requests_shouldServeCurrentFloor(elevatorState) {
				elevatorState, doorTimer = openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
				continue
			}
			db := requests_chooseDirection(elevatorState)
			applyDecision(db, &elevatorState, elevatorStateCh)

		case <-floorTicker.C: // alt av heis logikk

			sensorFloor := elevio.GetFloor()

			if sensorFloor != -1 {
				updateFloor(sensorFloor, &elevatorState, elevatorStateCh)
			}

			if doorTimer == nil && elevatorState.Floor != -1 && elevatorState.Behaviour == EB_Moving {
				elevatorState, doorTimer = clearFloorRequests(elevatorState, elevatorStateCh)
			}

			if elevatorState.Behaviour != EB_DoorOpen && requests_checkForRequests(elevatorState) {
				db := requests_chooseDirection(elevatorState)
				applyDecision(db, &elevatorState, elevatorStateCh)
			}

		case <-stopBtnCh:
			fmt.Println(elevatorState.Requests)
		}

	}
}

func applyDecision(db DirnBehaviourPair, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	// Guard: never publish an impossible "moving away from end floor" state.
	if (elevatorState.Floor == 0 && db.ElevatorBehaviour == EB_Moving && db.Dirn == D_Down) ||
		(elevatorState.Floor == N_FLOORS-1 && db.ElevatorBehaviour == EB_Moving && db.Dirn == D_Up) {
		db = DirnBehaviourPair{Dirn: D_Stop, ElevatorBehaviour: EB_Idle}
	}

	if db.ElevatorBehaviour == EB_DoorOpen {
		elevio.SetDoorOpenLamp(true)
	} else {
		elevio.SetDoorOpenLamp(false)
	}
	elevio.SetMotorDirection(elevio.MotorDirection(db.Dirn))

	if elevatorState.Dirn == db.Dirn && elevatorState.Behaviour == db.ElevatorBehaviour {
		return
	}
	elevatorState.Dirn = db.Dirn
	elevatorState.Behaviour = db.ElevatorBehaviour
	sendState(elevatorState, elevatorStateCh)
}

func clearFloorRequests(elevatorState ElevatorState, elevatorStateCh chan ElevatorState) (ElevatorState, <-chan time.Time) {
	if requests_shouldServeCurrentFloor(elevatorState) {
		return openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
	}

	return elevatorState, nil
}

func openDoorAndClearCurrentFloor(elevatorState ElevatorState, elevatorStateCh chan ElevatorState) (ElevatorState, <-chan time.Time) {
	elevio.SetMotorDirection(elevio.MD_Stop)

	elevatorState = requests_clearAtCurrentFloor(elevatorState)
	updateBehaviourAndRequests(EB_DoorOpen, elevatorState.Requests, &elevatorState, elevatorStateCh)

	return elevatorState, time.After(3000 * time.Millisecond)
}

func serveCurrentFloorNow(elevatorState ElevatorState, elevatorStateCh chan ElevatorState) (ElevatorState, <-chan time.Time, bool) {
	if elevatorState.Floor == -1 {
		return elevatorState, nil, false
	}
	if elevatorState.Behaviour == EB_DoorOpen {
		return elevatorState, nil, false
	}
	if !requests_shouldServeCurrentFloor(elevatorState) {
		return elevatorState, nil, false
	}

	elevatorState, doorTimer := openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
	return elevatorState, doorTimer, true
}

//////Gammel//////
/*
func FSM2(assignerToFsmCh chan [4][3]bool, elevatorStateCh chan ElevatorState, stopBtnCh chan bool) {

	elevatorState := InitElevatorState()

	InitElevator(&elevatorState, elevatorStateCh)

	floorTicker := time.NewTicker(50 * time.Millisecond)
	defer floorTicker.Stop()

	fmt.Println("FSM started") //TO DO: Fjern denne
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
				updateRequests(merged, &elevatorState, elevatorStateCh)

				if elevatorState.Floor == -1 || doorTimer != nil {
					continue
				}

				db := requests_chooseDirection(elevatorState)
				if db.ElevatorBehaviour == EB_DoorOpen {
					elevio.SetMotorDirection(elevio.MD_Stop)
					elevio.SetDoorOpenLamp(true)
					elevatorState = requests_clearAtCurrentFloor(elevatorState)
					updateRequests(elevatorState.Requests, &elevatorState, elevatorStateCh)
					updateBehaviour(EB_DoorOpen, &elevatorState, elevatorStateCh)
					doorTimer = time.After(3000 * time.Millisecond)
				} else {
					if db.Dirn != elevatorState.Dirn {
						updateDirection(db.Dirn, &elevatorState, elevatorStateCh)
						elevio.SetMotorDirection(elevio.MotorDirection(db.Dirn))
					}
					updateBehaviour(db.ElevatorBehaviour, &elevatorState, elevatorStateCh)
				}
			}

		case <-doorTimer:
			elevio.SetDoorOpenLamp(false)
			doorTimer = nil

			db := requests_chooseDirection(elevatorState)
			updateDirection(db.Dirn, &elevatorState, elevatorStateCh)
			updateBehaviour(db.ElevatorBehaviour, &elevatorState, elevatorStateCh)
			elevio.SetMotorDirection(elevio.MotorDirection(db.Dirn))

		case <-floorTicker.C:
			sensorFloor := elevio.GetFloor()
			if sensorFloor != -1 {
				updateFloor(sensorFloor, &elevatorState, elevatorStateCh)
			}

			if doorTimer != nil || elevatorState.Floor == -1 || elevatorState.Behaviour != EB_Moving {
				continue
			}

			if requests_shouldStop(elevatorState) {
				elevio.SetMotorDirection(elevio.MD_Stop)
				elevio.SetDoorOpenLamp(true)
				elevatorState = requests_clearAtCurrentFloor(elevatorState)
				updateRequests(elevatorState.Requests, &elevatorState, elevatorStateCh)
				updateDirection(D_Stop, &elevatorState, elevatorStateCh)
				updateBehaviour(EB_DoorOpen, &elevatorState, elevatorStateCh)
				doorTimer = time.After(3000 * time.Millisecond)
				continue
			}

			db := requests_chooseDirection(elevatorState)
			if db.Dirn != elevatorState.Dirn {
				updateDirection(db.Dirn, &elevatorState, elevatorStateCh)
				elevio.SetMotorDirection(elevio.MotorDirection(db.Dirn))
			}

		case <-stopBtnCh:
			fmt.Println(elevatorState.Requests)

		}
	}
}
*/