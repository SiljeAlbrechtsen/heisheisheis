package fsm

type DirnBehaviourPair struct {
	Dirn              Direction
	ElevatorBehaviour Behaviour
}

// A-Sjekker om det er noen bestillinger i systemet
func requests_checkForRequests(e ElevatorState) bool {
	for f := 0; f < N_FLOORS; f++ {
		for b := 0; b < N_BUTTONS; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

// requests_above returns true if there are any requests above current floor
func requests_above(e ElevatorState) bool {
	for f := e.Floor + 1; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

// requests_below returns true if there are any requests below current floor
func requests_below(e ElevatorState) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

// requests_here returns true if there are any requests at current floor
func requests_here(e ElevatorState) bool {
	for btn := 0; btn < N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

// requests_chooseDirection (exact C logic)
func requests_chooseDirection(e ElevatorState) DirnBehaviourPair {
	switch e.Dirn {
	case D_Up:
		if requests_above(e) {
			return DirnBehaviourPair{D_Up, EB_Moving}
		} else if requests_here(e) {
			return DirnBehaviourPair{D_Down, EB_DoorOpen}
		} else if requests_below(e) {
			return DirnBehaviourPair{D_Down, EB_Moving}
		}
		return DirnBehaviourPair{D_Stop, EB_Idle}

	case D_Down:
		if requests_below(e) {
			return DirnBehaviourPair{D_Down, EB_Moving}
		} else if requests_here(e) {
			return DirnBehaviourPair{D_Up, EB_DoorOpen}
		} else if requests_above(e) {
			return DirnBehaviourPair{D_Up, EB_Moving}
		}
		return DirnBehaviourPair{D_Stop, EB_Idle}

	case D_Stop:
		if requests_here(e) {
			return DirnBehaviourPair{D_Stop, EB_DoorOpen}
		} else if requests_above(e) {
			return DirnBehaviourPair{D_Up, EB_Moving}
		} else if requests_below(e) {
			return DirnBehaviourPair{D_Down, EB_Moving}
		}
		return DirnBehaviourPair{D_Stop, EB_Idle}

	default:
		return DirnBehaviourPair{D_Stop, EB_Idle}
	}
}

// requests_shouldStop (exact C logic)
func requests_shouldStop(e ElevatorState) bool {

	if e.Behaviour == EB_DoorOpen {
		return true
	} //A-TO DO: Sjekk om denne er kanskje unødvendig

	switch e.Dirn {
	case D_Down:
		return e.Requests[e.Floor][B_HallDown] ||
			e.Requests[e.Floor][B_Cab] ||
			!requests_below(e) //A-Sjekker om det er bestilling i cab, hall, eller under oss

	case D_Up:
		return e.Requests[e.Floor][B_HallUp] ||
			e.Requests[e.Floor][B_Cab] ||
			!requests_above(e) //A-Sjekker om det er bestilling i cab, hall, eller over oss

	case D_Stop:
		fallthrough
	default:
		return true //Defaul så stopper heisen
	}
}

// requests_shouldClearImmediately (exact C logic)
func requests_shouldClearImmediately(e ElevatorState, btnFloor int, btnType Button) bool {
	return e.Floor == btnFloor &&
		((e.Dirn == D_Up && btnType == B_HallUp) ||
			(e.Dirn == D_Down && btnType == B_HallDown) ||
			e.Dirn == D_Stop ||
			btnType == B_Cab)
}

func requests_shouldServeCurrentFloor(e ElevatorState) bool {
	switch e.Dirn {
	case D_Up:
		return e.Requests[e.Floor][B_HallUp] ||
			e.Requests[e.Floor][B_Cab] ||
			(!requests_above(e) && e.Requests[e.Floor][B_HallDown])

	case D_Down:
		return e.Requests[e.Floor][B_HallDown] ||
			e.Requests[e.Floor][B_Cab] ||
			(!requests_below(e) && e.Requests[e.Floor][B_HallUp])

	case D_Stop:
		return requests_here(e)

	default:
		return false
	}
}

// requests_clearAtCurrentFloor (exact C logic, returns new Elevator)
func requests_clearAtCurrentFloor(e ElevatorState) ElevatorState {
	e.Requests[e.Floor][B_Cab] = false
	switch e.Dirn {
	case D_Up:
		if !requests_above(e) && !e.Requests[e.Floor][B_HallUp] {
			e.Requests[e.Floor][B_HallDown] = false
		}
		e.Requests[e.Floor][B_HallUp] = false

	case D_Down:
		if !requests_below(e) && !e.Requests[e.Floor][B_HallDown] {
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