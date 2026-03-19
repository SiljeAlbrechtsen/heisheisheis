package fsm

import (
	"time"

	elevio "Project/Driver"
	hardware "Project/Hardware"
	t "Project/types"
)

const doorOpenDuration = 3000 * time.Millisecond
const errorTimeout = 3000 * time.Millisecond

func RunElevator(worldviewToFSMCh chan t.Worldview, elevatorStateCh chan ElevatorState, printHallOrdersReqCh chan bool) {

	obstruct := false

	elevatorState := InitElevatorState()
	InitElevator(&elevatorState, elevatorStateCh)

	// Periodisk ticker for fallback-bevegelse og etasjeoppdatering
	floorTicker := time.NewTicker(200 * time.Millisecond)
	defer floorTicker.Stop()

	// Etasjesensor med 20ms polling
	floorSensorCh := make(chan int)
	go elevio.PollFloorSensor(floorSensorCh)

	var doorTimer <-chan time.Time
	errorTimer := time.NewTimer(errorTimeout)
	defer stopAndDrainTimer(errorTimer)

	stopBtnCh := make(chan bool)
	obstructCh := make(chan bool)
	errorLightCh := make(chan bool, 1)
	go elevio.PollStopButton(stopBtnCh)
	go elevio.PollObstructionSwitch(obstructCh)
	go hardware.ErrorLight(errorLightCh)

	
	for {
		select {

		case worldview := <-worldviewToFSMCh:
			mergedRequests := mergeRequestsFromWorldview(elevatorState.Requests, worldview)
			requestsChanged := elevatorState.Requests != mergedRequests
			elevatorState.Requests = mergedRequests

			if shouldServeCurrentFloor(elevatorState) {
				elevatorState, doorTimer = openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
				continue
			}
			if elevatorCanMove(elevatorState) {
				elevatorState, doorTimer = executeMovementPlan(elevatorState, elevatorStateCh)
			} else if requestsChanged {
				sendState(&elevatorState, elevatorStateCh)
			}

		case <-doorTimer:
			doorTimer = nil
			// Hold døren åpen så lenge obstruction er aktiv eller heisen er i error
			if obstruct || elevatorState.Error {
				doorTimer = time.After(doorOpenDuration)
				continue
			}
			if shouldServeCurrentFloor(elevatorState) {
				elevatorState, doorTimer = openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
				continue
			}
			elevatorState, doorTimer = executeMovementPlan(elevatorState, elevatorStateCh)

		case <-floorTicker.C:
			if !(obstruct && elevatorState.Behaviour == EB_DoorOpen) && elevio.GetFloor() != -1 {
				sendLatestBool(errorLightCh, updateErrorState(false, &elevatorState, elevatorStateCh))
				resetTimer(errorTimer, errorTimeout)
			}
			// Fallback: start bevegelse hvis heisen har bestillinger men ikke beveger seg
			if elevatorCanMove(elevatorState) {
				elevatorState, doorTimer = executeMovementPlan(elevatorState, elevatorStateCh)
			}

		case floor := <-floorSensorCh:
			updateFloor(floor, &elevatorState, elevatorStateCh)

			if doorTimer == nil && elevatorState.Behaviour == EB_Moving {
				elevatorState, doorTimer = clearFloorRequests(elevatorState, elevatorStateCh)
			}
			if elevatorCanMove(elevatorState) {
				elevatorState, doorTimer = executeMovementPlan(elevatorState, elevatorStateCh)
			}

		case <-stopBtnCh:
			sendLatestBool(printHallOrdersReqCh, true)

		case obstruct = <-obstructCh:
			// Rydd error-state umiddelbart når obstruction fjernes
			if !obstruct {
				sendLatestBool(errorLightCh, updateErrorState(false, &elevatorState, elevatorStateCh))
				resetTimer(errorTimer, errorTimeout)
			}

		case <-errorTimer.C:
			sendLatestBool(errorLightCh, updateErrorState(true, &elevatorState, elevatorStateCh))
		}
	}
}

// InitElevator kjører heisen til nærmeste etasje og nullstiller all state.
func InitElevator(elevator *ElevatorState, elevatorStateCh chan ElevatorState) {
	hardware.TurnOffAllLights()

	if elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		for elevio.GetFloor() == -1 {
			time.Sleep(50 * time.Millisecond)
		}
		elevio.SetMotorDirection(elevio.MD_Stop)
	}

	elevator.Floor = elevio.GetFloor()
	sendState(elevator, elevatorStateCh)
}

// mergeRequestsFromWorldview synkroniserer FSM sine requests med worldview sine bekreftede ordrer.
// Inkluderer alle Confirmed hall-ordrer denne heisen eier, og alle cab-ordrer.
func mergeRequestsFromWorldview(currentRequests [N_FLOORS][N_BUTTONS]bool, worldview t.Worldview) [N_FLOORS][N_BUTTONS]bool {
	requests := currentRequests

	for f := 0; f < N_FLOORS; f++ {
		upOrder := worldview.HallOrders[f][B_HallUp]
		if upOrder.SyncState == t.Confirmed && upOrder.OwnerID == worldview.IdElevator {
			requests[f][B_HallUp] = true
		}
		downOrder := worldview.HallOrders[f][B_HallDown]
		if downOrder.SyncState == t.Confirmed && downOrder.OwnerID == worldview.IdElevator {
			requests[f][B_HallDown] = true
		}
	}

	if worldview.AllCabOrders != nil {
		for f := 0; f < N_FLOORS; f++ {
			if worldview.AllCabOrders[worldview.IdElevator][f] {
				requests[f][B_Cab] = true
			}
		}
	}

	return requests
}

// executeMovementPlan velger retning og utfører enten døråpning eller bevegelse.
func executeMovementPlan(e ElevatorState, ch chan ElevatorState) (ElevatorState, <-chan time.Time) {
	db := chooseDirection(e)
	if db.ElevatorBehaviour == EB_DoorOpen {
		return openDoorAndClearCurrentFloor(e, ch)
	}
	applyDecision(db, &e, ch)
	return e, nil
}

func applyDecision(db DirnBehaviourPair, elevatorState *ElevatorState, elevatorStateCh chan ElevatorState) {
	// Forhindre at heisen kjører ut over første eller siste etasje
	if (elevatorState.Floor == 0 && db.ElevatorBehaviour == EB_Moving && db.Dirn == D_Down) ||
		(elevatorState.Floor == N_FLOORS-1 && db.ElevatorBehaviour == EB_Moving && db.Dirn == D_Up) {
		db = DirnBehaviourPair{Dirn: D_Stop, ElevatorBehaviour: EB_Idle}
	}

	elevio.SetDoorOpenLamp(db.ElevatorBehaviour == EB_DoorOpen)
	elevio.SetMotorDirection(elevio.MotorDirection(db.Dirn))

	if elevatorState.Dirn == db.Dirn && elevatorState.Behaviour == db.ElevatorBehaviour {
		return
	}
	elevatorState.Dirn = db.Dirn
	elevatorState.Behaviour = db.ElevatorBehaviour
	sendState(elevatorState, elevatorStateCh)
}

// clearFloorRequests åpner døren og fjerner ordrer hvis heisen er på en etasje med aktive ordrer.
// GetFloor() != -1 forhindrer clearing når heisen er mellom etasjer.
func clearFloorRequests(elevatorState ElevatorState, elevatorStateCh chan ElevatorState) (ElevatorState, <-chan time.Time) {
	if shouldServeCurrentFloor(elevatorState) && elevio.GetFloor() != -1 && !elevatorState.Error {
		return openDoorAndClearCurrentFloor(elevatorState, elevatorStateCh)
	}
	return elevatorState, nil
}

func openDoorAndClearCurrentFloor(elevatorState ElevatorState, elevatorStateCh chan ElevatorState) (ElevatorState, <-chan time.Time) {
	elevio.SetMotorDirection(elevio.MD_Stop)
	elevatorState = clearAtCurrentFloor(elevatorState)
	elevatorState.Behaviour = EB_DoorOpen
	elevio.SetDoorOpenLamp(true)
	sendState(&elevatorState, elevatorStateCh)

	return elevatorState, time.After(doorOpenDuration)
}

func elevatorCanMove(e ElevatorState) bool {
	return e.Behaviour != EB_DoorOpen && !e.Error && hasAnyRequests(e) && elevio.GetFloor() != -1
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
