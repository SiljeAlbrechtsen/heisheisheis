<<<<<<< HEAD:Project/assignment.go
package assignment
=======
assignment

package main 


>>>>>>> origin/main:Project/assignment/assignment.go

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strconv"
)

<<<<<<< HEAD:Project/assignment.go
// TODO: Finne ut hvordan vi skal sende det over. Det må være public, men er det dårlig praksis? Burde vi heller sende over en public kopi?
// TODO: Nå blir id som tall. Må den endres til one, two osv eller funker det?
// TODO: Alexsey: Elevator state? Hvordan er den? Stemmer formatet med JSON filen?
// TODO: Eventuelt ta JSON pakking i egen modul? Hva er sammenhengen med assignment her?

// TODO: Dårlig kodekvalitet å bruke myID i alle?

// Bytte navn?
type hallRequestsInputJSON struct { 
	HallRequests [NumFloors][Directions]bool // TODO: Bytte navn på directions til NumDirections?
	States       map[string]stateInputJSON 
}

type stateInputJSON  struct {
	Behaviour   string           
	Floor       int              
	Direction   string           
	CabRequests [NumFloors]bool  
}

// Hjelpefunksjon
func buildState(state StateElevator) stateInputJSON{
	 return stateInputJSON{
        Behaviour:   state.Behaviour,
        Floor:       state.Floor,
        Direction:   state.Direction,
        CabRequests: state.MyCabOrders,
    }
}

// Hjelpefunksjon
func convertHallOrdersToBool(hallOrders hallOrders) [NumFloors][Directions]bool {
	var converted [NumFloors][Directions]bool

	for f := 0; f < NumFloors; f++ {
		for d := 0; d < Directions; d++ {
			converted[f][d] = hallOrders[f][d] == Confirmed
		}
	}
	return converted
}

// Hjelpefunksjon
func buildInputHallRequestAssigner(latestWorldviews map[int]Worldview, myID int) hallRequestsInputJSON {
    // Hent hall requests fra egen worldview
    hallRequests := convertHallOrdersToBool(latestWorldviews[myID].hallOrders)

    states := make(map[string]stateInputJSON)
    for id, worldview := range convertWordlviewToJSON latestWorldviews {
        states[strconv.Itoa(id)] = buildState(worldview.State)
    }

    return hallRequestsInputJSON{
        HallRequests: hallRequests,
        States:       states,
    }
}

func convertWorldviewToJSON(latestWorldviews map[int]Worldview, myID int) ([]byte, error) {
    input := buildInputHallRequestAssigner(latestWorldviews, myID)
    return json.MarshalIndent(input, "", "\t")
}


func assignHallRequests(latestWorldviews map[int]Worldview, myID int) (map[string][][]bool, error) {
	jsonInput, err := convertWorldviewToJSON(latestWorldview, myID)
	if err != nil {
		return nil, err
	}

	// Sende til hall request assigner og få svar
	cmd := exec.Command("./hall_request_assigner")
	cmd.Stdin = bytes.NewReader(jsonInput)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Pakke ut JSON. Evt i annen funk?
	var result map[string][][]bool
	err = json.Unmarshal(output, &result)

	return result, nil 
}

// Etterpå, lage en testfunksjon som tester assignment også fikse channels og fikse sånn at worldview sender en kopi av worldview inn


// Jeg får inn channel med worldview. Den skal brukes til å

// Lager en funksjon som skal kalle på 
/*
Tanke videre:
I en annen funksjon så kan den ta inn channel og få inn worldview.
Bruke hjelpefunksjon til å få det på JSON også bare kjøre assigner funk
Må neste gang finne ut hvordan jeg kan faktisk bruke assigner funksjonen. 
*/














/*
func behaviourToString(behaviour Behaviour) string {
	switch behaviour {
	case Idle:
		return "idle"
	case Moving:
		return "moving"
	case DoorOpen:
		return "doorOpen"
	default:
		return "idle"
	}
}


func directionToString(dir Direction) string {
	switch dir {
	case Up:
		return "up"
	case Down:
		return "down"
	case Stop:
		return "stop"
	default:
		return "stop"
	}
}


func buildInputHallRequestAssigner(latestWorldviews map[int]Worldview) inputHallRequestsAssigner {
	var input inputHallRequestsAssigner

	input.stateInput = make(map[string]stateInputJSON)

	first := true

	// ID ? hvorfor bruker jeg ikke den?
	for id, w := range latestWorldviews {
		if first {
			input.HallRequests = hallOrdersToBool(w.hallOrders)
			first = false
		}
		idToStr := "id_" + strconv.Itoa(id)

		input.stateInput[id].Behaviour = behaviourToString(w.State.Behaviour)
		input.stateInput[id].Direction = directionToString(w.State.Direction)
		input.stateInput[id].Floor = w.State.Floor // TODO: Alexsey Hvor ligger floor lagret?
		input.stateInput[id].CabRequests = w.MyCabOrders
	}
	return input
}


// Funk som konverterer til json. får inn channel fra worldview.
// kjøre den kostfunksjonen
// channel som sender til fsm




*/


=======

//____________________________________________________________________________________________________________________
//---------------------- CHANNELS ------------------------------------------------------------------------------------
//____________________________________________________________________________________________________________________
>>>>>>> origin/main:Project/assignment/assignment.go






/*
Input:
Hall requests 
Elevator states 
Elevator ID (local)

*/



// ----------------BOSS-----------------

/*
 
import (
	"fmt"
	// Må importere heiskode som har funksjonene chooseDirection og shouldStop
)




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


<<<<<<< HEAD:Project/assignment.go
*/
=======
/*
Lage funksjon som pakker json fil
exec.Command
hraExecutable

*/
>>>>>>> origin/main:Project/assignment/assignment.go
