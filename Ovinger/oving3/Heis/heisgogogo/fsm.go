import (
	"fmt"
)

func setAllLights(es Elevator) {
    for floor := 0; floor < N_FLOORS; floor++ {
        for btn := 0; btn < N_BUTTONS; btn++ {
            elevatorRequestButtonLight(
                floor,
                btn,
                es.requests[floor][btn],
            )
        }
    }
}



func fsm_onInitBetweenFloors(e *Elevator) {
	elevatorMotorDirection(D_Down)
	e.dir = D_Down
	e.behavior = EB_Moving
}