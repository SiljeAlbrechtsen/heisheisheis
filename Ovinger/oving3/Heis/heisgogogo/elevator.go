import (
	"ftm"
)

// Enums deklarasjon
type ElevatorBehaviour int
type Dir int
type Button int

// Konstanter
const (
    EB_Idle ElevatorBehaviour = iota
    EB_DoorOpen
    EB_Moving
)

const (
    D_Up Dir = iota
    D_Down
    D_Stop
)

const (
    B_HallUp Button = iota
    B_HallDown
    B_Cab
)

// Struct
type Elevator struct {
    floor     int
    dirn      Dirn
    requests  [N_FLOORS][N_BUTTONS]bool
    behaviour ElevatorBehaviour

    config struct {
        doorOpenDuration_s float64
    }
}


func elevatorBehaviourToString(eb ElevatorBehaviour) string {
    switch eb {
    case EB_Idle:
        return "EB_Idle"
    case EB_DoorOpen:
        return "EB_DoorOpen"
    case EB_Moving:
        return "EB_Moving"
    default:
        return "EB_UNDEFINED"
    }
}

func elevatorDirnToString(d Dir) string {
    switch d {
    case D_Up:
        return "D_Up"
    case D_Down:
        return "D_Down"
    case D_Stop:
        return "D_Stop"
    default:
        return "D_UNDEFINED"
    }
}

func elevatorButtonToString(b Button) string {
    switch b {
    case B_HallUp:
        return "B_HallUp"
    case B_HallDown:
        return "B_HallDown"
    case B_Cab:
        return "B_Cab"
    default:
        return "B_UNDEFINED"
    }
}


func elevatorPrint(es Elevator) {
    fmt.Println("  +--------------------+")
    fmt.Printf(
        "  |floor = %-2d          |\n"+
            "  |dirn  = %-12.12s|\n"+
            "  |behav = %-12.12s|\n",
        es.floor,
        elevatorDirnToString(es.dirn),
        elevatorBehaviourToString(es.behaviour),
    )
    fmt.Println("  +--------------------+")
    fmt.Println("  |  | up  | dn  | cab |")

    for f := N_FLOORS - 1; f >= 0; f-- {
        fmt.Printf("  | %d", f)
        for btn := 0; btn < N_BUTTONS; btn++ {
            if (f == N_FLOORS-1 && btn == B_HallUp) ||
                (f == 0 && btn == B_HallDown) {
                fmt.Print("|     ")
            } else {
                if es.requests[f][btn] {
                    fmt.Print("|  #  ")
                } else {
                    fmt.Print("|  -  ")
                }
            }
        }
        fmt.Println("|")
    }
    fmt.Println("  +--------------------+")
}

func elevatorUninitialized() Elevator {
    elevatorHardwareInit()

    var e Elevator
    e.floor = -1
    e.dirn = D_Stop
    e.behaviour = EB_Idle
    e.config.doorOpenDuration_s = 3.0

    return e
}

func elevatorFloorSensor() int {
    return elevatorHardwareGetFloorSensorSignal()
}

func elevatorRequestButton(f int, b Button) bool {
    return elevatorHardwareGetButtonSignal(b, f)
}

func elevatorStopButton() bool {
    return elevatorHardwareGetStopSignal()
}

func elevatorObstruction() bool {
    return elevatorHardwareGetObstructionSignal()
}

func elevatorFloorIndicator(f int) {
    elevatorHardwareSetFloorIndicator(f)
}

func elevatorRequestButtonLight(f int, b Button, v bool) {
    elevatorHardwareSetButtonLamp(b, f, v)
}

func elevatorDoorLight(v bool) {
    elevatorHardwareSetDoorOpenLamp(v)
}

func elevatorStopButtonLight(v bool) {
    elevatorHardwareSetStopLamp(v)
}

func elevatorMotorDirection(d Dirn) {
    elevatorHardwareSetMotorDirection(d)
}
