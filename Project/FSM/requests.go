package fsm

import elev "Project/elevator"

type DirnBehaviourPair struct {
	Dirn              Direction
	ElevatorBehaviour Behaviour
}

func hasAnyRequests(e ElevatorState) bool {
	for f := 0; f < elev.N_FLOORS; f++ {
		for b := 0; b < elev.N_BUTTONS; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func requestsAbove(e ElevatorState) bool {
	for f := e.Floor + 1; f < elev.N_FLOORS; f++ {
		for btn := 0; btn < elev.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsBelow(e ElevatorState) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < elev.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsHere(e ElevatorState) bool {
	if e.Floor < 0 || e.Floor >= elev.N_FLOORS {
		return false
	}
	for btn := 0; btn < elev.N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func chooseDirection(e ElevatorState) DirnBehaviourPair {
	switch e.Dirn {
	case elev.D_Up:
		if requestsAbove(e) {
			return DirnBehaviourPair{Dirn: elev.D_Up, ElevatorBehaviour: elev.EB_Moving}
		} else if requestsHere(e) {
			return DirnBehaviourPair{Dirn: elev.D_Down, ElevatorBehaviour: elev.EB_DoorOpen}
		} else if requestsBelow(e) {
			return DirnBehaviourPair{Dirn: elev.D_Down, ElevatorBehaviour: elev.EB_Moving}
		}
		return DirnBehaviourPair{Dirn: elev.D_Stop, ElevatorBehaviour: elev.EB_Idle}

	case elev.D_Down:
		if requestsBelow(e) {
			return DirnBehaviourPair{Dirn: elev.D_Down, ElevatorBehaviour: elev.EB_Moving}
		} else if requestsHere(e) {
			return DirnBehaviourPair{Dirn: elev.D_Up, ElevatorBehaviour: elev.EB_DoorOpen}
		} else if requestsAbove(e) {
			return DirnBehaviourPair{Dirn: elev.D_Up, ElevatorBehaviour: elev.EB_Moving}
		}
		return DirnBehaviourPair{Dirn: elev.D_Stop, ElevatorBehaviour: elev.EB_Idle}

	case elev.D_Stop:
		if requestsHere(e) {
			return DirnBehaviourPair{Dirn: elev.D_Stop, ElevatorBehaviour: elev.EB_DoorOpen}
		} else if requestsAbove(e) {
			return DirnBehaviourPair{Dirn: elev.D_Up, ElevatorBehaviour: elev.EB_Moving}
		} else if requestsBelow(e) {
			return DirnBehaviourPair{Dirn: elev.D_Down, ElevatorBehaviour: elev.EB_Moving}
		}
		return DirnBehaviourPair{Dirn: elev.D_Stop, ElevatorBehaviour: elev.EB_Idle}

	default:
		return DirnBehaviourPair{Dirn: elev.D_Stop, ElevatorBehaviour: elev.EB_Idle}
	}
}

func shouldServeCurrentFloor(e ElevatorState) bool {
	if e.Floor < 0 {
		return false
	}
	switch e.Dirn {
	case elev.D_Up:
		return e.Requests[e.Floor][elev.B_HallUp] ||
			e.Requests[e.Floor][elev.B_Cab] ||
			(!requestsAbove(e) && e.Requests[e.Floor][elev.B_HallDown])

	case elev.D_Down:
		return e.Requests[e.Floor][elev.B_HallDown] ||
			e.Requests[e.Floor][elev.B_Cab] ||
			(!requestsBelow(e) && e.Requests[e.Floor][elev.B_HallUp])

	case elev.D_Stop:
		return requestsHere(e)

	default:
		return false
	}
}

func clearAtCurrentFloor(e ElevatorState) ElevatorState {
	if e.Floor < 0 || e.Floor >= elev.N_FLOORS {
		return e
	}
	e.Requests[e.Floor][elev.B_Cab] = false
	switch e.Dirn {
	case elev.D_Up:
		if !requestsAbove(e) && !e.Requests[e.Floor][elev.B_HallUp] {
			e.Requests[e.Floor][elev.B_HallDown] = false
		}
		e.Requests[e.Floor][elev.B_HallUp] = false

	case elev.D_Down:
		if !requestsBelow(e) && !e.Requests[e.Floor][elev.B_HallDown] {
			e.Requests[e.Floor][elev.B_HallUp] = false
		}
		e.Requests[e.Floor][elev.B_HallDown] = false

	case elev.D_Stop:
		fallthrough
	default:
		e.Requests[e.Floor][elev.B_HallUp] = false
		e.Requests[e.Floor][elev.B_HallDown] = false
	}
	return e
}
