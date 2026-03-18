package fsm

import (
	"fmt"
	"time"

	elevio "Project/Driver"
	hardware "Project/Hardware"
)

func FSM3(assignerToFsmCh chan [4][3]bool, elevatorStateCh chan ElevatorState, printHallOrdersReqCh chan bool) {

	obstruct := false

	elevatorState := InitElevatorState()
	InitElevator(&elevatorState, elevatorStateCh)

	// Cab-lys ticker — kun for periodisk oppdatering av cab-lys
	floorTicker := time.NewTicker(500 * time.Millisecond)
	defer floorTicker.Stop()

	// Etasjesensor med 20ms polling — håndterer all etasjelogikk
	floorSensorCh := make(chan int)
	go elevio.PollFloorSensor(floorSensorCh)

	var doorTimer <-chan time.Time
	doorTimer = nil
	errorTimer := time.NewTimer(5 * time.Second)
	defer stopAndDrainTimer(errorTimer)

	stopBtnCh := make(chan bool)
	obstructCh := make(chan bool)
	errorLightCh := make(chan bool, 1)
	go elevio.PollStopButton(stopBtnCh)
	go elevio.PollObstructionSwitch(obstructCh)
	go hardware.ErrorLight(errorLightCh)

	for {
		select {
		case newRequests := <-assignerToFsmCh:
			fmt.Printf("*********\n%f\n*********\n", newRequests)
			mergedState := updateElevatorRequests(elevatorState, newRequests)
			updateRequests(mergedState.Requests, &elevatorState, elevatorStateCh) // TODO: Fjerne?

			if requests_shouldServeCurrentFloor(elevatorState) {
				elevatorState, doorTimer = openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
				fmt.Println("###\nServing current floor immediately after receiving new requests\n###")
				continue
			}
			// Start bevegelse umiddelbart ved nye bestillinger
			if elevatorCanMove(elevatorState) {
				db := requests_chooseDirection(elevatorState)
				applyDecision(db, &elevatorState, elevatorStateCh)
			}

		case <-doorTimer:
			doorTimer = nil
			// Hold døren åpen så lenge obstruction er aktiv eller heisen er i error
			if obstruct || elevatorState.Error {
				doorTimer = time.After(3000 * time.Millisecond)
				continue
			}
			if requests_shouldServeCurrentFloor(elevatorState) {
				elevatorState, doorTimer = openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
				continue
			}
			db := requests_chooseDirection(elevatorState)
			applyDecision(db, &elevatorState, elevatorStateCh)

		case <-floorTicker.C:
			refreshCabLights(elevatorState)
			if !(obstruct && elevatorState.Behaviour == EB_DoorOpen) && elevio.GetFloor() != -1 {
				sendLatestBool(errorLightCh, updateErrorState(false, &elevatorState, elevatorStateCh))
				resetTimer(errorTimer, 5*time.Second)
			}
			// Fallback: start bevegelse hvis heisen har bestillinger men ikke beveger seg
			if elevatorCanMove(elevatorState) {
				db := requests_chooseDirection(elevatorState)
				applyDecision(db, &elevatorState, elevatorStateCh)
			}

		case floor := <-floorSensorCh:
			updateFloor(floor, &elevatorState, elevatorStateCh)

			if doorTimer == nil && elevatorState.Behaviour == EB_Moving {
				elevatorState, doorTimer = clearFloorRequests(elevatorState, elevatorStateCh)
			}
			if elevatorCanMove(elevatorState) {
				db := requests_chooseDirection(elevatorState)
				applyDecision(db, &elevatorState, elevatorStateCh)
			}

		case <-stopBtnCh:
			fmt.Println(elevatorState)
			sendLatestBool(printHallOrdersReqCh, true)

		case obstruct = <-obstructCh:
			// Rydd error-state umiddelbart når obstruction fjernes
			if !obstruct {
				sendLatestBool(errorLightCh, updateErrorState(false, &elevatorState, elevatorStateCh))
				resetTimer(errorTimer, 5*time.Second)
			}

		case <-errorTimer.C:
			fmt.Println("Tiden er ute!")
			sendLatestBool(errorLightCh, updateErrorState(true, &elevatorState, elevatorStateCh))
			fmt.Println(elevatorState)
		}
	}
}

// Init for elevator
func InitElevator(elevator *ElevatorState, elevatorStateCh chan ElevatorState) {

	hardware.TurnOffAllLights() //A-Fjerner alle gamle lys, i tilfelle heisen starter med noen lys på

	if elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		for elevio.GetFloor() == -1 {
			time.Sleep(50 * time.Millisecond) //TO DO: HARD CONSTANT fjernt
		}
		elevio.SetMotorDirection(elevio.MD_Stop)
	}

	elevator.Floor = elevio.GetFloor()
	sendState(elevator, elevatorStateCh)
}

// Helper functions for FSM3
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
	if requests_shouldServeCurrentFloor(elevatorState) && elevio.GetFloor() != -1 && !elevatorState.Error { //A-La til elevio.GetFloor() != -1 for å unngå å clear'e requests når heisen er mellom etasjer
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

func elevatorCanMove(e ElevatorState) bool {
	if e.Behaviour != EB_DoorOpen && !e.Error && requests_checkForRequests(e) && elevio.GetFloor() != -1 {
		return true
	}
	return false
}

func resetTimer(t *time.Timer, d time.Duration) {
	stopAndDrainTimer(t)
	t.Reset(d)
}

func stopAndDrainTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

func sendLatestBool(ch chan bool, v bool) {
	select {
	case ch <- v:
	default:
		select {
		case <-ch:
		default:
		}
		ch <- v
	}
}

func refreshCabLights(e ElevatorState) {
	for f := 0; f < N_FLOORS; f++ {
		if e.Requests[f][B_Cab] {
			elevio.SetButtonLamp(elevio.BT_Cab, f, true)
		} else {
			elevio.SetButtonLamp(elevio.BT_Cab, f, false)
		}
	}
}
