package main

// requests_above returns true if there are any requests above current floor
func requests_above(e Elevator) bool {
	for f := e.floor + 1; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.requests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

// requests_below returns true if there are any requests below current floor
func requests_below(e Elevator) bool {
	for f := 0; f < e.floor; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.requests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

// requests_here returns true if there are any requests at current floor
func requests_here(e Elevator) bool {
	for btn := 0; btn < N_BUTTONS; btn++ {
		if e.requests[e.floor][btn] != 0 {
			return true
		}
	}
	return false
}

// requests_chooseDirection (exact C logic)
func requests_chooseDirection(e Elevator) DirnBehaviourPair {
	switch e.dirn {
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

	case D_Stop: // there should only be one request in the Stop case
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
func requests_shouldStop(e Elevator) bool {
	switch e.dirn {
	case D_Down:
		return e.requests[e.floor][B_HallDown] != 0 ||
			e.requests[e.floor][B_Cab] != 0 ||
			!requests_below(e)

	case D_Up:
		return e.requests[e.floor][B_HallUp] != 0 ||
			e.requests[e.floor][B_Cab] != 0 ||
			!requests_above(e)

	case D_Stop:
		fallthrough
	default:
		return true
	}
}

// requests_shouldClearImmediately (exact C logic)
func requests_shouldClearImmediately(e Elevator, btnFloor int, btnType Button) bool {
	return e.floor == btnFloor &&
		((e.dirn == D_Up && btnType == B_HallUp) ||
			(e.dirn == D_Down && btnType == B_HallDown) ||
			e.dirn == D_Stop ||
			btnType == B_Cab)
}

// requests_clearAtCurrentFloor (exact C logic, returns new Elevator)
func requests_clearAtCurrentFloor(e Elevator) Elevator {
	e.requests[e.floor][B_Cab] = 0
	switch e.dirn {
	case D_Up:
		if !requests_above(e) && e.requests[e.floor][B_HallUp] == 0 {
			e.requests[e.floor][B_HallDown] = 0
		}
		e.requests[e.floor][B_HallUp] = 0

	case D_Down:
		if !requests_below(e) && e.requests[e.floor][B_HallDown] == 0 {
			e.requests[e.floor][B_HallUp] = 0
		}
		e.requests[e.floor][B_HallDown] = 0

	case D_Stop:
		fallthrough
	default:
		e.requests[e.floor][B_HallUp] = 0
		e.requests[e.floor][B_HallDown] = 0
	}
	return e
}
