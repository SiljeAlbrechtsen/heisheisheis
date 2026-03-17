package hardware

import (
	elevio "Project/Driver"
	"fmt"
	"time"
)

var errorLightCh chan bool

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

func LightsListener(lightOnCh chan [2]int, lightsOffCh chan [2]int) { //To DO: Fjern printksjon som tar inn en bool for on/off

	for {
		select {
		case a := <-lightOnCh:
			fmt.Println("--------------\nLIGHT ON: ", a, "\n---------------")
			elevio.SetButtonLamp(elevio.ButtonType(a[1]), a[0], true)
		case a := <-lightsOffCh:
			fmt.Println("--------------\nLIGHT OFF: ", a, "\n---------------")
			elevio.SetButtonLamp(elevio.ButtonType(a[1]), a[0], false)
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
