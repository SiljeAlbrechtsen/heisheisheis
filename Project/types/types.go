package types

import (
	elevio "Project/Driver"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

const N_FLOORS = 4
const N_BUTTONS = 3

type Button int

const (
	B_HallUp Button = iota
	B_HallDown
	B_Cab
)

type Behaviour int

const (
	EB_Idle Behaviour = iota
	EB_DoorOpen
	EB_Moving
)

type Direction int

const (
	D_Down Direction = -1
	D_Stop Direction = 0
	D_Up   Direction = 1
)

type ElevatorState struct {
	Floor     int
	Dirn      Direction
	Behaviour Behaviour
	Requests  [N_FLOORS][N_BUTTONS]bool
	Error     bool
	config    struct {
		doorOpenDuration_s float64
	}
}

type HallRequestsMatrix [4][3]bool

func InitDriver() {
	addr := resolveElevatorAddr()
	elevio.Init(addr, N_FLOORS)
}

func InitElevatorState() ElevatorState {
	InitDriver()
	return ElevatorState{
		Floor:     -1,
		Dirn:      D_Stop,
		Behaviour: EB_Idle,
		Error:     false,
		config: struct {
			doorOpenDuration_s float64
		}{doorOpenDuration_s: 3.0},
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