package hardware

import (
	elevio "Project/Driver"
	elev "Project/elevator"
	wv "Project/worldview"
	"time"
)

func ButtonsListener(cabButtonCh chan int, hallButtonCh chan [2]int) {
	elevioButtonCh := make(chan elevio.ButtonEvent)
	go elevio.PollButtons(elevioButtonCh)

	for {
		select {
		case event := <-elevioButtonCh:
			if event.Button == elevio.BT_Cab {
				cabButtonCh <- event.Floor
			} else {
				hallButtonCh <- [2]int{event.Floor, int(event.Button)}
			}
		}
	}
}

// TurnOffAllLights slår av alle knapplys, dørlampe og stopplampe ved oppstart.
func TurnOffAllLights() {
	for f := 0; f < elev.N_FLOORS; f++ {
		for b := elevio.ButtonType(0); b < elevio.ButtonType(elev.N_BUTTONS); b++ {
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

func ButtonLightsListener(lightsCh <-chan wv.Worldview) {
	for worldview := range lightsCh {
		// Hall-lys: på hvis ordren er Confirmed
		for f := 0; f < elev.N_FLOORS; f++ {
			for d := 0; d < 2; d++ {
				elevio.SetButtonLamp(elevio.ButtonType(d), f, worldview.HallOrders[f][d].SyncState == wv.Confirmed)
			}
		}
		// Cab-lys: på hvis cab-ordre finnes for denne heisen
		if cabOrders, ok := worldview.AllCabOrders[worldview.IdElevator]; ok {
			for f := 0; f < elev.N_FLOORS; f++ {
				elevio.SetButtonLamp(elevio.BT_Cab, f, cabOrders[f])
			}
		}
	}
}
