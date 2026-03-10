package main

//cabButtonCh
//hallButtonCh

import (
	elevio "Project/Driver"
	"fmt"
)

func ButtonsListener(cabButtonCh chan int, hallButtonCh chan elevio.ButtonEvent) {

	elevioButtonCh := make(chan elevio.ButtonEvent)
	go elevio.PollButtons(elevioButtonCh)

	for {
		select {
		case a := <-elevioButtonCh:
			if a.Button == elevio.BT_Cab {
				cabButtonCh <- a.Floor
			} else {
				hallButtonCh <- a
			}
			fmt.Println(a)
		}
	}
}
