package main

//"fmt"  //Begge brukes i test main
//"time" //
//
//elevio "../Driver"

func main() {

	elevatorState := InitElevatorState()

	InitElevator(&elevatorState)

	elevatorState.requests = [N_FLOORS][N_BUTTONS]bool{
		{false, false, true},
		{false, false, false},
		{false, true, false},
		{true, false, false},
	}

	ClearFloorRequest(elevatorState.requests, &elevatorState)

	PrintElevatorState(elevatorState)

}
