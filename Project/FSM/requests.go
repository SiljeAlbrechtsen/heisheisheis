package fsm

import elevio "Project/Driver"

type DirnBehaviourPair struct {
	Dirn              Direction
	ElevatorBehaviour Behaviour
}

func hasAnyRequests(e ElevatorState) bool {
	for f := 0; f < N_FLOORS; f++ {
		for b := 0; b < N_BUTTONS; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func requestsAbove(e ElevatorState) bool {
	for f := e.Floor + 1; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsBelow(e ElevatorState) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsHere(e ElevatorState) bool {
	if e.Floor < 0 || e.Floor >= N_FLOORS {
		return false
	}
	for btn := 0; btn < N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func chooseDirection(e ElevatorState) DirnBehaviourPair {
	switch e.Dirn {
	case D_Up:
		if requestsAbove(e) {
			return DirnBehaviourPair{Dirn: D_Up, ElevatorBehaviour: EB_Moving}
		} else if requestsHere(e) {
			return DirnBehaviourPair{Dirn: D_Down, ElevatorBehaviour: EB_DoorOpen}
		} else if requestsBelow(e) {
			return DirnBehaviourPair{Dirn: D_Down, ElevatorBehaviour: EB_Moving}
		}
		return DirnBehaviourPair{Dirn: D_Stop, ElevatorBehaviour: EB_Idle}

	case D_Down:
		if requestsBelow(e) {
			return DirnBehaviourPair{Dirn: D_Down, ElevatorBehaviour: EB_Moving}
		} else if requestsHere(e) {
			return DirnBehaviourPair{Dirn: D_Up, ElevatorBehaviour: EB_DoorOpen}
		} else if requestsAbove(e) {
			return DirnBehaviourPair{Dirn: D_Up, ElevatorBehaviour: EB_Moving}
		}
		return DirnBehaviourPair{Dirn: D_Stop, ElevatorBehaviour: EB_Idle}

	case D_Stop:
		if requestsHere(e) {
			return DirnBehaviourPair{Dirn: D_Stop, ElevatorBehaviour: EB_DoorOpen}
		} else if requestsAbove(e) {
			return DirnBehaviourPair{Dirn: D_Up, ElevatorBehaviour: EB_Moving}
		} else if requestsBelow(e) {
			return DirnBehaviourPair{Dirn: D_Down, ElevatorBehaviour: EB_Moving}
		}
		return DirnBehaviourPair{Dirn: D_Stop, ElevatorBehaviour: EB_Idle}

	default:
		return DirnBehaviourPair{Dirn: D_Stop, ElevatorBehaviour: EB_Idle}
	}
}

func shouldServeCurrentFloor(e ElevatorState) bool {
	if elevio.GetFloor() == -1 {
		return false
	}
	switch e.Dirn {
	case D_Up:
		return e.Requests[e.Floor][B_HallUp] ||
			e.Requests[e.Floor][B_Cab] ||
			(!requestsAbove(e) && e.Requests[e.Floor][B_HallDown])

	case D_Down:
		return e.Requests[e.Floor][B_HallDown] ||
			e.Requests[e.Floor][B_Cab] ||
			(!requestsBelow(e) && e.Requests[e.Floor][B_HallUp])

	case D_Stop:
		return requestsHere(e)

	default:
		return false
	}
}

func clearAtCurrentFloor(e ElevatorState) ElevatorState {
	if e.Floor < 0 || e.Floor >= N_FLOORS {
		return e
	}
	e.Requests[e.Floor][B_Cab] = false
	switch e.Dirn {
	case D_Up:
		if !requestsAbove(e) && !e.Requests[e.Floor][B_HallUp] {
			e.Requests[e.Floor][B_HallDown] = false
		}
		e.Requests[e.Floor][B_HallUp] = false

	case D_Down:
		if !requestsBelow(e) && !e.Requests[e.Floor][B_HallDown] {
			e.Requests[e.Floor][B_HallUp] = false
		}
		e.Requests[e.Floor][B_HallDown] = false

	case D_Stop:
		fallthrough
	default:
		e.Requests[e.Floor][B_HallUp] = false
		e.Requests[e.Floor][B_HallDown] = false
	}
	return e
}
