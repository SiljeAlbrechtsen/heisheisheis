package types

import (
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
}

type OrderSyncState int

const (
	None OrderSyncState = iota
	Unconfirmed
	Confirmed
	DeleteProposed
)

type Order struct {
	SyncState OrderSyncState
	OwnerID   string
}

type HallOrders [N_FLOORS][2]Order

// Worldview type — used across FSM, worldview, and assignment packages
type Worldview struct {
	IdElevator   string
	HallOrders   HallOrders
	State        ElevatorState
	AllCabOrders map[string][N_FLOORS]bool
	ErrorState   bool // Settes ved motorstopp/obstruction
	Dead         bool // Settes ved nettverkstap
}

func InitElevatorState() ElevatorState {
	return ElevatorState{
		Floor:     -1,
		Dirn:      D_Stop,
		Behaviour: EB_Idle,
		Error:     false,
	}
}

func ResolveElevatorAddr() string {
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
