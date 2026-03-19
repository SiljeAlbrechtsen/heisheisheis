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

func buildState(id string, worldview wv.Worldview) stateInputJSON {
	return stateInputJSON{
		Behaviour:   behaviourToString(worldview.State.Behaviour),
		Floor:       worldview.State.Floor,
		Direction:   directionToString(worldview.State.Dirn),
		CabRequests: worldview.AllCabOrders[id],
	}
}

func convertHallOrdersToBool(hallOrders wv.HallOrders) [wv.NumFloors][wv.Directions]bool {
	var converted [wv.NumFloors][wv.Directions]bool

	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			order := hallOrders[f][d]
			// Reassign orders that do not have a living owner
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

	var result map[string]wv.AssignmentMatrix
	err = json.Unmarshal(output, &result)

	return result, nil
}

func RunHallRequestAssigner(
	myID string,
	worldviewsForAssignerCh <-chan map[string]wv.Worldview,
	assignmentCh chan<- map[string]wv.AssignmentMatrix,
) {
	var lastResult map[string]wv.AssignmentMatrix
	for {
		latestWorldviews := <-worldviewsForAssignerCh
		result, err := assignHallRequests(latestWorldviews, myID)
		if err != nil {
			fmt.Println("Assigner feil:", err)
			continue
		}
		if !reflect.DeepEqual(result, lastResult) {
			assignmentCh <- result
			lastResult = result
		}
	}
}
