package main

import(
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"Driver-go/elevio"
)

type Elevator struct {
	floor      int
	dirn       Direction
	behaviour  Behaviour
	requests   [N_FLOORS][N_BUTTONS]int
	config     struct {
		doorOpenDuration_s float64
	}
}

func resolveElevatorAddr() string {
	if addr := strings.TrimSpace(os.Getenv("ELEVATOR_ADDR")); addr != "" {
		return addr
	}
	candidates := []string{"localhost:15657"}
	if out, err := exec.Command("sh", "-c", "ip route | awk '/default/ {print $3}'").Output(); err == nil {
		ip := strings.TrimSpace(string(out))
		if ip != "" {
			candidates = append(candidates, ip+":15657")
		}
	}
	for _, addr := range candidates {
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return addr
		}
	}
	return candidates[0]
}

func elevator_uninitialized() Elevator {
	addr := resolveElevatorAddr()
	elevio.Init(addr, 4) //(addr, N_FLOORS)
	return Elevator{
		floor:     -1,
		dirn:      0,
		behaviour: 0,
		config: struct {
			doorOpenDuration_s float64
		}{doorOpenDuration_s: 3.0},
	}
}

func main(){
	fmt.Println("Started!")

	e := elevator_uninitialized()


	for {
		lights_on()
		time.Sleep(200 * time.Millisecond)
	}
	
}