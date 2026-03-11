package hardware

//cabButtonCh
//hallButtonCh

import (
	elevio "Project/Driver"
	"fmt"
)

func ButtonsListener(cabButtonCh chan int, hallButtonCh chan [2]int) {

	elevioButtonCh := make(chan elevio.ButtonEvent)
	go elevio.PollButtons(elevioButtonCh)

	for {
		select {
		case a := <-elevioButtonCh:
			if a.Button == elevio.BT_Cab {
				cabButtonCh <- a.Floor
			} else {
				result := [2]int{a.Floor, int(a.Button)}
				hallButtonCh <- result
			}
			fmt.Println(a)
		}
	}
}
