package assignment

import (
	wv "Project/worldview"
	fsm "Project/FSM"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
)

// TODO: Dårlig kodekvalitet å bruke myID i alle?

// Bytte navn?
type hallRequestsInputJSON struct {
	HallRequests [wv.NumFloors][wv.Directions]bool `json:"hallRequests"`
	States       map[string]stateInputJSON          `json:"states"`
}

type stateInputJSON struct {
	Behaviour   string            `json:"behaviour"`
	Floor       int               `json:"floor"`
	Direction   string            `json:"direction"`
	CabRequests [wv.NumFloors]bool `json:"cabRequests"`
}

func behaviourToString(b fsm.Behaviour) string {
	switch b {
	case fsm.EB_Moving:
		return "moving"
	case fsm.EB_DoorOpen:
		return "doorOpen"
	default:
		return "idle"
	}
}

func directionToString(d fsm.Direction) string {
	switch d {
	case fsm.D_Up:
		return "up"
	case fsm.D_Down:
		return "down"
	default:
		return "stop"
	}
}

// Hjelpefunksjon
func buildState(worldview wv.Worldview) stateInputJSON {
	return stateInputJSON{
		Behaviour:   behaviourToString(worldview.State.Behaviour),
		Floor:       worldview.State.Floor,
		Direction:   directionToString(worldview.State.Dirn),
		CabRequests: worldview.MycabOrders,
	}
}

// Hjelpefunksjon
func convertHallOrdersToBool(hallOrders wv.HallOrders) [wv.NumFloors][wv.Directions]bool {
	var converted [wv.NumFloors][wv.Directions]bool
	orderNotAssigned := false

	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			if hallOrders[f][d].SyncState == wv.Confirmed && hallOrders[f][d].OwnerID == wv.NoOwner {
				orderNotAssigned = true
			}
			// Reassigner bare orders som ikke har noen owner :)
			converted[f][d] = orderNotAssigned
			orderNotAssigned = false
		}
	}
	return converted
}

// Hjelpefunksjon
func buildInputHallRequestAssigner(latestWorldviews map[string]wv.Worldview, MyID string) hallRequestsInputJSON {
	// Hent hall requests fra egen worldview
	hallRequests := convertHallOrdersToBool(latestWorldviews[MyID].HallOrders)

	states := make(map[string]stateInputJSON)
	for id, worldview := range latestWorldviews {
		states[id] = buildState(worldview)
	}

	return hallRequestsInputJSON{
		HallRequests: hallRequests,
		States:       states,
	}
}

func convertWorldviewToJSON(latestWorldviews map[string]wv.Worldview, MyID string) ([]byte, error) {
	input := buildInputHallRequestAssigner(latestWorldviews, MyID)
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// Eller bare assignRequests, siden den sier noe om caborders også?
func assignHallRequests(latestWorldviews map[string]wv.Worldview, MyID string) (map[string][4][3]bool, error) {
	jsonInput, err := convertWorldviewToJSON(latestWorldviews, MyID)
	if err != nil {
		return nil, err
	}

	// Sende til hall request assigner og få svar
	var stderr bytes.Buffer
	//fmt.Println("JSON sendt til assigner:", string(jsonInput))
	cmd := exec.Command("./assignment/hall_request_assigner")
	cmd.Stdin = bytes.NewReader(jsonInput)
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("JSON sendt til assigner:", string(jsonInput))
		fmt.Println("Binær stderr:", stderr.String())
		fmt.Println("Binær-feil:", err)
		return nil, err
	}

	// Pakke ut JSON. Evt i annen funk?
	var result map[string][4][3]bool
	err = json.Unmarshal(output, &result)

	return result, nil
}

// GO routine
func RunHallRequestAssigner(
	myID string,
	worldviewToAssignerCh <-chan map[string]wv.Worldview,
	assignerToFsmCh chan<- [4][3]bool,
	assignerToWordviewCh chan<- map[string][4][3]bool,
) {
	var lastResult map[string][4][3]bool
	for {
		latestWorldviews := <-worldviewToAssignerCh
		//fmt.Println("Assigner: mottok worldview")
		result, err := assignHallRequests(latestWorldviews, myID)
		if err != nil {
			fmt.Println("Assigner feil:", err)
			continue
		}
		fmt.Println("Assigner: sender til FSM:", result[myID])
		assignerToFsmCh <- result[myID]
		// reflect.DeepEqual er med i standard bib. i go og brukes for å sammenligne maps.
		if !reflect.DeepEqual(result, lastResult) {
			assignerToWordviewCh <- result
			lastResult = result
		}
	}
}
