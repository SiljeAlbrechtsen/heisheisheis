package assignment 

import (
	"fmt"
	// Må importere heiskode som har funksjonene chooseDirection og shouldStop
)

/*
Input:
Hall requests 
Elevator states 
Elevator ID (local)

*/



// Array-størrelse må være kjent i go ved kompileringstid.
const (
	NFloors  = 4
	NButtons = 3
) // Er det samme som å bare skrive const før de, men dette er mer lesbart

// Stor bokstav for public. Altså andre package kan bruke de. Evt endre
type Elevator struct {
	Floor 	int
	Dirn  	Dirn 
	Behaviour Behaviour
	Requests  [NFloors][NButtons]bool // De må være definert før. Kompileringstid
}

type OnCleared func(btn int, floor int) // istedenfor Function pointer
// Kan brukes til å skru av knappelys, sende melding til andre noder, oppdatere worldview ++

// Den antar at vi har bestemt at elevator e skal stoppe i denne etasjen. Fjerner så relevante requests
func clearAtCurrentFloor(e Elevator, onCleared OnCleared) Elevator {
	for btn := 0; btn < NButtons; btn++ {
		if e.Requests[e.Floor][btn] {
			e.Requests[e.Floor][btn] = false
			if onCleared != nil { 
				onCleared(btn, e.Floor) 
			}
		}
	}
	return e
} // Husk! Denne tar inn en kopi, dermed må vi bruke den slik 
// e = cleatAtCurrentFloor(e, nil)



const (
	TravelTime      = 2500 // Tatt fra config.d
	DoopOpenTime    = 3000
)

// enums
type Behaviour int
const (
	EB_Idle Behaviour = iota
	EB_Moving 
	EB_DoorOpen
)

type Dirn int
const (
	D_Down Dirn = -1
	D_Stop Dirn = 0
	D_up   Dirn = 1
)

// Funksjonene vi allerede skal ha:
// requests_ChooseDirection(e Elevator ) Dirn // Her velger heisen retning basert på requests
// requests_shouldStop(e Elevator) Bool // Om vi skal stoppe i etasjen. Fra enkelheis logikken

// kostfunksjonen
func TimeToIdle(e Elevator) int {
	duration := 0 

	switch e.Behaviour {
	
		// Hvis heisen står stille skal choose direction velge retning basert på requests
	case EB_Idle:
		e.Dirn = Requests_ChooseDirection(e)
		if e.Dirn == D_Stop { // Hvis det ikke er noen requests skal den stå stille
			return duration 
		}
	// Justerer door time døren allerede er åpen
	case EB_DoorOpen:
		duration += TRAVEL_TIME / 2 
		e.floor += int(e.Dirn)
	}

	// simulerer helt til heisen blir tom. Altså heis = idle. 
	for {
		if requests_shouldStop(e) {
			e = Requests_clearAtCurrentFloor(e, nil) // legg inn funksjon for nil
			duration += DOOR_OPEN_TIME

			e.Dirn = Requests_ChooseDirection(e)
			if e.dirn == D_Stop {
                return duration 
            }
		}
	}
	e.Floor += int(e.Dirn)
	duration += TravelTime
}


