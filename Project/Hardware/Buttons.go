package hardware

import (
	elevio "Project/Driver"
	t "Project/types"
	"fmt"
	"time"
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


func TurnOffAllLights() { //A-La til en slå av alle lys ed init
	for f := 0; f < 4; f++ { //A-TO DO: Fjern hardkoding
		for b := elevio.ButtonType(0); b < 3; b++ {
			elevio.SetButtonLamp(b, f, false)
		}
	}
	elevio.SetDoorOpenLamp(false)
	elevio.SetStopLamp(false)
}

func ErrorLight(errorLight chan bool) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	blinking := false
	lampOn := false

	for {
		select {
		case val := <-errorLight:
			blinking = val
			if !blinking {
				elevio.SetStopLamp(false)
			}

		case <-ticker.C:
			if blinking {
				lampOn = !lampOn
				elevio.SetStopLamp(lampOn)
			}
		}
	}
}

func ButtonLightsListener(lightsCh <-chan t.Worldview) {
	for wv := range lightsCh {
		// Hall-lys: på hvis ordren er Confirmed
		for f := 0; f < t.N_FLOORS; f++ {
			for d := 0; d < 2; d++ {
				elevio.SetButtonLamp(elevio.ButtonType(d), f, wv.HallOrders[f][d].SyncState == t.Confirmed)
			}
		}
		// Cab-lys: på hvis cab-ordre finnes for denne heisen
		if cabOrders, ok := wv.AllCabOrders[wv.IdElevator]; ok {
			for f := 0; f < t.N_FLOORS; f++ {
				elevio.SetButtonLamp(elevio.BT_Cab, f, cabOrders[f])
			}
		}
	}
}
