package main

import (
	"time"
)

// getWallTime returns the current wall-clock time in seconds as float64,
// similar to the C version using gettimeofday.
func getWallTime() float64 {
	now := time.Now()
	return float64(now.Unix()) + float64(now.Nanosecond())/1e9
}

var (
	timerEndTime float64
	timerActive  bool
)

// TimerStart starts a timer for the given duration in seconds.
func TimerStart(duration float64) {
	timerEndTime = getWallTime() + duration
	timerActive = true
}

// TimerStop stops the timer.
func TimerStop() {
	timerActive = false
}

// TimerTimedOut returns true if the timer is active and has timed out.
func TimerTimedOut() bool {
	return timerActive && getWallTime() > timerEndTime
}
