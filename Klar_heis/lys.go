package main

import "time"

import "Driver-go/elevio"

func lights_on(){
	if elevio.GetStop() {
		elevio.SetStopLamp(true)
		time.Sleep(1 * time.Second)
		elevio.SetStopLamp(false)
	}
}