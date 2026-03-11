package fsm

//run
//go run ./FSM

//"fmt"  //Begge brukes i test main
//"time" //
//
//elevio "../Driver"
import (
	
)

func DebugRun() {

	elevatorState := InitElevatorState()

	InitElevator(&elevatorState)

	elevatorState.Requests = [N_FLOORS][N_BUTTONS]bool{
		{false, false, true},
		{false, false, false},
		{false, true, false},
		{true, false, false},
	}

	//ClearFloorRequest(elevatorState.Requests, &elevatorState)

	PrintElevatorState(elevatorState)

}
