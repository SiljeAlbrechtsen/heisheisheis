// Use `go run foo.go` to run your program

package main

import (
    . "fmt"
    "runtime"
   // "time"
)

var i = 0

// Bruker struct for vi bare skal sende signal og ikke data. 
func server(inc <-chan struct{}, dec <-chan struct{}, get <-chan struct{}, reply chan<- int) {
	i := 0
	// Denne funksjonen håndterer i nå
	for { // while
		select {
		case <-inc:
			i++
		case <-dec:
			i--
		case <-get:
			reply <- i
			return // stopper når ferdig
		}
	}
}

func incrementing(inc chan<- struct{}, done chan<- struct{}) {
    //TODO: increment i 1000000 times
	for j := 0; j < 1_000_000; j++ {
		//i++   
		inc <- struct{}{}
	}
	done <- struct{}{}
}

func decrementing(dec chan<- struct{}, done chan<- struct{}) {
    //TODO: decrement i 1000000 times
	for j := 0; j < 999_999; j++ {
		//i--
		dec <- struct{}{}
	}
	done <- struct{}{}
}

func main() {
    // What does GOMAXPROCS do? What happens if you set it to 1?
    runtime.GOMAXPROCS(2)    // Hvor mange tråder. 
	
    // Lager channels til server
	incCh := make(chan struct{})
	decCh := make(chan struct{})

	getCh := make(chan struct{})
	replyCh := make(chan int)

	done := make(chan struct{})

	// Starter server. Den holder styr på i
	go server(incCh, decCh, getCh, replyCh)

	// Starter workers. work baby, work
	go incrementing(incCh, done)
	go decrementing(decCh, done)

	// venter på workers. wait
	<-done
	<-done

	// Henter verdien fra server
	getCh <- struct{}{}
//	result := <-replyCh

    // TODO: Spawn both functions as goroutines
	//go incrementing()
	//go decrementing()
	
    // We have no direct way to wait for the completion of a goroutine (without additional synchronization of some sort)
    // We will do it properly with channels soon. For now: Sleep.
    //time.Sleep(500*time.Millisecond)
    Println("The magic number is (go):", <-replyCh)
}