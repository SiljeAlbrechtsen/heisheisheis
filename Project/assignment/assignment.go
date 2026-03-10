package assignment

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strconv"
	wv "Project/worldview"
)

// TODO: Dårlig kodekvalitet å bruke myID i alle?

// Bytte navn?
type hallRequestsInputJSON struct { 
	HallRequests [wv.NumFloors][wv.Directions]bool // TODO: Bytte navn på directions til NumDirections?
	States       map[string]stateInputJSON 
}

type stateInputJSON  struct {
	Behaviour   string           
	Floor       int              
	Direction   string           
	CabRequests [wv.NumFloors]bool  
}

// Hjelpefunksjon
func buildState(worldview wv.Worldview) stateInputJSON{
	 return stateInputJSON{
        Behaviour:   worldview.state.Behaviour,
        Floor:       worldview.state.Floor,
        Direction:   worldview.state.Direction,
        CabRequests: worldview.mycabOrders, 
    }
}

// Hjelpefunksjon
func convertHallOrdersToBool(hallOrders wv.HallOrders) [wv.NumFloors][wv.Directions]bool {
	var converted [wv.NumFloors][wv.Directions]bool

	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			converted[f][d] = (hallOrders[f][d].syncState == wv.Confirmed)
		}
	}
	return converted
}

// Hjelpefunksjon
func buildInputHallRequestAssigner(latestWorldviews map[int]wv.Worldview, MyID int) hallRequestsInputJSON {
    // Hent hall requests fra egen worldview
    hallRequests := convertHallOrdersToBool(latestWorldviews[MyID].hallOrders)

    states := make(map[string]stateInputJSON)
    for id, worldview := range latestWorldviews {
        states[strconv.Itoa(id)] = buildState(worldview)
    }

    return hallRequestsInputJSON{
        HallRequests: hallRequests,
        States:       states,
    }
}

func convertWorldviewToJSON(latestWorldviews map[int]wv.Worldview, MyID int) ([]byte, error) {
    input := buildInputHallRequestAssigner(latestWorldviews, MyID)
    return json.MarshalIndent(input, "", "\t")
}

// TODO: Ligger hall_request_assigner i riktig mappe?
// Eller bare assignRequests, siden den sier noe om caborders også?
func assignHallRequests(latestWorldviews map[int]wv.Worldview, MyID int) (map[string][][]bool, error) {
	jsonInput, err := convertWorldviewToJSON(latestWorldviews, MyID)
	if err != nil {
		return nil, err
	}

	// Sende til hall request assigner og få svar
	cmd := exec.Command("./Project/assignment/hall_request_assigner")
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



// GO routine
func RunHallRequestAssigner(
	myID int, 
	worldviewToAssignerCh <-chan map[int]wv.Worldview, 
	assignerToFsmCh chan<- [][]bool,
	) {
    for {
        latestWorldviews := <- worldviewToAssignerCh
        result, err := assignHallRequests(latestWorldviews, myID)
        if err != nil {
            continue
        }
        assignerToFsmCh <- result[strconv.Itoa(myID)]
    }
}
