package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Started!")

	e := elevator_uninitialized()
	inputPollRateMs := 25

	ConLoad("elevator.con", func(key, val string) {
		switch key {
		case "dooropenduration_s":
			fmt.Sscanf(val, "%f", &e.config.doorOpenDuration_s)
		case "inputpollrate_ms":
			fmt.Sscanf(val, "%d", &inputPollRateMs)
		}
	})

	if elevator_floorSensor() == -1 {
		FsmOnInitBetweenFloors(&e)
	}

	prevButtons := [N_FLOORS][N_BUTTONS]int{}
	prevFloor := -1
	prevStop := 0
	prevObstruction := 0

	for {
		stop := elevator_stopButton()
		if stop != prevStop {
			elevator_stopButtonLight(stop)
			if stop != 0 {
				elevator_motorDirection(D_Stop)
				e.dirn = D_Stop
				for f := 0; f < N_FLOORS; f++ {
					for b := 0; b < N_BUTTONS; b++ {
						e.requests[f][b] = 0
					}
				}
				setAllLights(e)
				TimerStop()
			} else {
				if e.floor != -1 {
					e.behaviour = EB_DoorOpen
					elevator_doorLight(1)
					TimerStart(e.config.doorOpenDuration_s)
				} else {
					FsmOnInitBetweenFloors(&e)
				}
			}
		}
		prevStop = stop
		if stop != 0 {
			time.Sleep(time.Duration(inputPollRateMs) * time.Millisecond)
			continue
		}

		obstruction := elevator_obstruction()
		if obstruction != prevObstruction {
			if obstruction != 0 {
				elevator_motorDirection(D_Stop)
			} else if e.behaviour == EB_Moving {
				elevator_motorDirection(e.dirn)
			}
		}
		prevObstruction = obstruction

		for f := 0; f < N_FLOORS; f++ {
			for b := 0; b < N_BUTTONS; b++ {
				v := elevator_requestButton(f, Button(b))
				if v != 0 && v != prevButtons[f][b] {
					FsmOnRequestButtonPress(&e, f, Button(b))
				}
				prevButtons[f][b] = v
			}
		}

		floor := elevator_floorSensor()
		if floor != -1 && floor != prevFloor {
			FsmOnFloorArrival(&e, floor)
		}
		prevFloor = floor

		if TimerTimedOut() {
			if obstruction != 0 {
				TimerStart(e.config.doorOpenDuration_s)
				time.Sleep(time.Duration(inputPollRateMs) * time.Millisecond)
				continue
			}
			TimerStop()
			FsmOnDoorTimeout(&e)
		}

		time.Sleep(time.Duration(inputPollRateMs) * time.Millisecond)
	}
}
