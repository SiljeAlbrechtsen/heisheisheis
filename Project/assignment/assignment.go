package assignment

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strconv"
)

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

// Eller bare assignRequests, siden den sier noe om caborders også?
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


// TODO: LEGG INN CHANNEL I MAIN
// GO routine
func runHallRequestAssignerEvery10ms(myID int, in <-chan map[int]Worldview, out chan<- [][]bool) {
	ticker := time.NewTicker(10 * time.Millisecond)
    defer ticker.Stop()

    var latestWorldviews map[int]Worldview

	for {
		select {
		case updatedWorldviews := in
		latestWorldview = updatedWorldviews

		case <-ticker.C:
			if latestWorldviews = nil {
				continue
			}
			result, err = assignHallRequests(latestWorldviews, myID)
			if err != nil {
                continue
            }
            out <- result[strconv.Itoa(myID)]
		}
	}
}



// Trenger egt ikke disse to funksjonene nå, det er implementert i runHallRequestAssignerEvery10ms
func receiveWorldviews(updatedWorldviewToAssignerCh <-chan map[int]Worldview) map[int]Worldview {
	receivedWorldviews := <-updatedWorldviewToAssignerCh
	return receivedWorldviews
}

func sendAssignedRequests(AssignedRequestsCh chan<- map[string][][]bool) {
	result, err = assignHallRequests(latestWorldviews, myID)
	if err != nil {
        // TODO: logg eller håndter feilen? Eller bare håndtere feilen?
        return
    }
    ch <- result[strconv.Itoa(myID)]
}

