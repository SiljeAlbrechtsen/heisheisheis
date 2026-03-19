package assignment

import (
	elev "Project/elevator"
	wv "Project/worldview"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
)

type hallRequestsInputJSON struct {
	HallRequests [elev.N_FLOORS][elev.N_DIRECTIONS]bool `json:"hallRequests"`
	States       map[string]stateInputJSON         `json:"states"`
}

type stateInputJSON struct {
	Behaviour   string             `json:"behaviour"`
	Floor       int                `json:"floor"`
	Direction   string             `json:"direction"`
	CabRequests [elev.N_FLOORS]bool `json:"cabRequests"`
}

func behaviourToString(b elev.Behaviour) string {
	switch b {
	case elev.EB_Moving:
		return "moving"
	case elev.EB_DoorOpen:
		return "doorOpen"
	default:
		return "idle"
	}
}

func directionToString(d elev.Direction) string {
	switch d {
	case elev.D_Up:
		return "up"
	case elev.D_Down:
		return "down"
	default:
		return "stop"
	}
}

func buildState(id string, worldview wv.Worldview) stateInputJSON {
	return stateInputJSON{
		Behaviour:   behaviourToString(worldview.State.Behaviour),
		Floor:       worldview.State.Floor,
		Direction:   directionToString(worldview.State.Dirn),
		CabRequests: worldview.AllCabOrders[id],
	}
}

func convertHallOrdersToBool(hallOrders wv.HallOrders) [elev.N_FLOORS][elev.N_DIRECTIONS]bool {
	var converted [elev.N_FLOORS][elev.N_DIRECTIONS]bool

	for f := 0; f < elev.N_FLOORS; f++ {
		for d := 0; d < elev.N_DIRECTIONS; d++ {
			order := hallOrders[f][d]
			// Reassign orders som ikke har en levende eier
			converted[f][d] = order.SyncState == wv.Confirmed &&
				(order.OwnerID == wv.NoOwner || order.OwnerID == wv.PeerDied)
		}
	}
	return converted
}

func buildAssignerInput(latestWorldviews map[string]wv.Worldview, myID string) hallRequestsInputJSON {
	hallRequests := convertHallOrdersToBool(latestWorldviews[myID].HallOrders)

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

func assignHallRequests(latestWorldviews map[string]wv.Worldview, myID string) (map[string]wv.AssignmentMatrix, error) {
	input := buildAssignerInput(latestWorldviews, myID)
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
	var result map[string]wv.AssignmentMatrix
	err = json.Unmarshal(output, &result)

	return result, nil
}

func RunHallRequestAssigner(
	myID string,
	worldviewToAssignerCh <-chan map[string]wv.Worldview,
	assignerToWorldviewCh chan<- map[string]wv.AssignmentMatrix,
) {
	var lastResult map[string]wv.AssignmentMatrix
	for {
		latestWorldviews := <-worldviewToAssignerCh
		result, err := assignHallRequests(latestWorldviews, myID)
		if err != nil {
			fmt.Println("Assigner feil:", err)
			continue
		}
		if !reflect.DeepEqual(result, lastResult) {
			assignerToWorldviewCh <- result
			lastResult = result
		}
	}
}
