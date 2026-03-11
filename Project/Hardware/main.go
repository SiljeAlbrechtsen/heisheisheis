package hardware

/*
func test99() {
	requests := [fsm.N_FLOORS][fsm.N_BUTTONS]bool{}

	cabButtonCh := make(chan int)
	hallButtonCh := make(chan elevio.ButtonEvent)

	go ButtonsListener(cabButtonCh, hallButtonCh)

	requestCh := make(chan [fsm.N_FLOORS][fsm.N_BUTTONS]bool, 1)
	elevatorStateCh := make(chan fsm.ElevatorState)

	go fsm.FSM2(requestCh, elevatorStateCh)

	for {
		select {
		case a := <-cabButtonCh:
			fmt.Printf("Cab button pressed at floor %d\n", a)
			requests = fsm.UpdateRequest(requests, a, elevio.BT_Cab)
			requestCh <- requests
			requests = [fsm.N_FLOORS][fsm.N_BUTTONS]bool{}
		case a := <-hallButtonCh:
			fmt.Printf("Hall button pressed: %+v\n", a)
			requests = fsm.UpdateRequest(requests, a.Floor, a.Button)
			requestCh <- requests
			requests = [fsm.N_FLOORS][fsm.N_BUTTONS]bool{}

		case b := <-elevatorStateCh:
			fmt.Printf("Elevator state updated: %+v\n", b)
		}
	}

}
*/
