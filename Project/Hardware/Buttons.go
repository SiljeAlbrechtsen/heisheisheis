package hardware

import (
	elevio "Project/Driver"
	t "Project/types"
	"time"
)

func ButtonsListener(cabRequestCh chan int, hallRequestCh chan [2]int) {
	elevioButtonCh := make(chan elevio.ButtonEvent)
	go elevio.PollButtons(elevioButtonCh)

	for {
		select {
		case event := <-elevioButtonCh:
			if event.Button == elevio.BT_Cab {
				cabRequestCh <- event.Floor
			} else {
				hallRequestCh <- [2]int{event.Floor, int(event.Button)}
			}
		}
	}
}

// TurnOffAllLights turns off all button lamps, the door lamp, and the stop lamp on startup.
func TurnOffAllLights() {
	for f := 0; f < t.N_FLOORS; f++ {
		for b := elevio.ButtonType(0); b < elevio.ButtonType(t.N_BUTTONS); b++ {
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

func ButtonLightsListener(lightStateCh <-chan t.Worldview) {
	for wv := range lightStateCh {
		// Hall lights: on if the order is Confirmed
		for f := 0; f < t.N_FLOORS; f++ {
			for d := 0; d < 2; d++ {
				elevio.SetButtonLamp(elevio.ButtonType(d), f, wv.HallOrders[f][d].SyncState == t.Confirmed)
			}
		}
		// Cab lights: on if a cab order exists for this elevator
		if cabOrders, ok := wv.AllCabOrders[wv.IdElevator]; ok {
			for f := 0; f < t.N_FLOORS; f++ {
				elevio.SetButtonLamp(elevio.BT_Cab, f, cabOrders[f])
			}
		}
	}
}
