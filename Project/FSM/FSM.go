package fsm

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
	obstructCh := make(chan bool)
	go elevio.PollStopButton(stopBtnCh)
	go elevio.PollObstructionSwitch(obstructCh)

	for { // 4 sjekk om det trenges å sjekke door open i should stop, 5 sjekk om det trenges å sjekke door open i should clear immediately
		// 6 Forenkle is setningene. kanskje en funksjon som sjekker om det skal bli tru eller ey
		select {
		case newRequests := <-assignerToFsmCh: //A-Tar i mot requests fra assigner og legger de i elevaterState, sender så oppdatering til worldview
			//fmt.Println(newRequests) //TO DO: FJERN
			mergedState := updateElevatorRequests(elevatorState, newRequests)
			updateRequests(mergedState.Requests, &elevatorState, elevatorStateCh)

			if doorTimer != nil && elevio.GetFloor() != -1 && requests_shouldServeCurrentFloor(elevatorState) { //A-Her skal sjekke om låv å betjene
				elevatorState, doorTimer = openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
				fmt.Println("###\nServing current floor immediately after receiving new requests\n###") //TO DO: FJERN
				continue
			}

			if doorTimer == nil {
				var served bool
				elevatorState, doorTimer, served = serveCurrentFloorNow(elevatorState, elevatorStateCh)
				if served { //A-om true på served så går vi ut av newrequest for løkken
					continue
				}
			}

			if elevatorState.Behaviour != EB_DoorOpen && requests_checkForRequests(elevatorState) {
				fmt.Println("\n*\n")
				db := requests_chooseDirection(elevatorState)
				applyDecision(db, &elevatorState, elevatorStateCh)
			}

		case <-doorTimer:
			doorTimer = nil
			if elevatorState.Error {
				fmt.Println("Y")
				doorTimer = time.After(3000 * time.Millisecond)
			}
			if requests_shouldServeCurrentFloor(elevatorState) {
				elevatorState, doorTimer = openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
				continue
			}
			db := requests_chooseDirection(elevatorState)
			applyDecision(db, &elevatorState, elevatorStateCh)

		case <-floorTicker.C: // alt av heis logikk

			if elevio.GetFloor() != -1 {
				updateFloor(elevio.GetFloor(), &elevatorState, elevatorStateCh)
			}

			if doorTimer == nil && elevio.GetFloor() != -1 && elevatorState.Behaviour == EB_Moving {
				elevatorState, doorTimer = clearFloorRequests(elevatorState, elevatorStateCh)
			}

			if elevatorState.Behaviour != EB_DoorOpen && requests_checkForRequests(elevatorState) && elevio.GetFloor() != -1 { //SJEKK SISTE
				db := requests_chooseDirection(elevatorState)
				applyDecision(db, &elevatorState, elevatorStateCh)
			}

		case <-stopBtnCh:
			fmt.Println(elevatorState.Requests)

		case obst := <-obstructCh: //A-Må kunn hente obstruction selv om den ikke er i åpen dør, eller mulig
			fmt.Println("Obstruction detected!       %f", obst)
			if doorTimer != nil && obst {
				updateErrorState(obst, &elevatorState, elevatorStateCh)
				fmt.Println("E_State!       %f", elevatorState.Error)
			}
			if doorTimer != nil && !obst {
				updateErrorState(obst, &elevatorState, elevatorStateCh)
				fmt.Println("E_State!       %f", elevatorState.Error)
				doorTimer = time.After(3000 * time.Millisecond)
			}
		}

	}
}

//Helper functions for FSM3

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

func serveCurrentFloorNow(elevatorState ElevatorState, elevatorStateCh chan ElevatorState) (ElevatorState, <-chan time.Time, bool) {
	if elevio.GetFloor() == -1 || elevatorState.Behaviour == EB_DoorOpen || !requests_shouldServeCurrentFloor(elevatorState) {
		return elevatorState, nil, false
	}

	elevatorState, doorTimer := openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
	return elevatorState, doorTimer, true
}

func elevatorCanMove(e ElevatorState) bool {

}
