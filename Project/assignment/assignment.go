package assignment

import (
	fsm "Project/FSM"
	wv "Project/worldview"
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
	States       map[string]stateInputJSON         `json:"states"`
}

type stateInputJSON struct {
	Behaviour   string             `json:"behaviour"`
	Floor       int                `json:"floor"`
	Direction   string             `json:"direction"`
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
func buildState(id string, worldview wv.Worldview) stateInputJSON {
	return stateInputJSON{
		Behaviour:   behaviourToString(worldview.State.Behaviour),
		Floor:       worldview.State.Floor,
		Direction:   directionToString(worldview.State.Dirn),
		CabRequests: worldview.AllCabOrders[id],
	}
}

// Hjelpefunksjon
func convertHallOrdersToBool(hallOrders wv.HallOrders) [wv.NumFloors][wv.Directions]bool {
	var converted [wv.NumFloors][wv.Directions]bool

	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			order := hallOrders[f][d]
			// Reassign orders som ikke har en levende eier
			converted[f][d] = order.SyncState == wv.Confirmed &&
				(order.OwnerID == wv.NoOwner || order.OwnerID == wv.PeerDied)
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
		if worldview.Dead || worldview.ErrorState {
			continue
		}
		states[id] = buildState(id, worldview)
	}

	return hallRequestsInputJSON{
		HallRequests: hallRequests,
		States:       states,
	}
}

// Eller bare assignRequests, siden den sier noe om caborders også?
func assignHallRequests(latestWorldviews map[string]wv.Worldview, MyID string) (map[string][4][3]bool, error) {
	input := buildInputHallRequestAssigner(latestWorldviews, MyID)
	if len(input.States) == 0 {
		return nil, fmt.Errorf("ingen tilgjengelige heiser (alle i error state)")
	}

	jsonInput, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	jsonInput = append(jsonInput, '\n')

	// Sende til hall request assigner og få svar
	var stderr bytes.Buffer
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
	//assignerToFsmCh chan<- [4][3]bool,
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
		// reflect.DeepEqual er med i standard bib. i go og brukes for å sammenligne maps.
		if !reflect.DeepEqual(result, lastResult) {
			//fmt.Println("Assigner: sender til FSM:", result[myID])
			//assignerToFsmCh <- result[myID]
			assignerToWordviewCh <- result
			lastResult = result
		}
	}
}
