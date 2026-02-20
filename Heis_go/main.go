package main

import (
	"./elevator"
)

func main() {
	e := elevator.Elevator{
		floor: 1,
		dirn:  elevator.D_Up,
		requests: [4][3]int{
			{0, 0, 0}, // floor 0
			{1, 0, 1}, // floor 1: HallUp + Cab
			{0, 1, 0}, // floor 2: HallDown
			{0, 0, 0}, // floor 3
		},
	}

	elevator.elevator_print(e)

	pair := elevator.requests_chooseDirection(e)
	fmt.Printf("chooseDirection: dir=%s, behaviour=%s\n",
		elevator.elevator_dirnToString(pair.dirn),
		elevator.elevator_behaviorToString(pair.behaviour))

	fmt.Printf("shouldStop: %v\n", elevator.requests_shouldStop(e))
}
