package synchronization

import (
	wv "Project/worldview"
	"fmt"
	"testing"
)

// makeOrder er en hjelpefunksjon for å opprette en Order
func makeOrder(state wv.OrderSyncState) wv.Order {
	return wv.Order{SyncState: state, OwnerID: ""}
}

// makeWorldview lager en minimal Worldview med gitte HallOrders
func makeWorldview(id string, hallOrders wv.HallOrders) wv.Worldview {
	return wv.Worldview{
		IdElevator:   id,
		HallOrders:   hallOrders,
		AllCabOrders: make(map[string][wv.NumFloors]bool),
	}
}

// detectUnconfirmedVsDeleteProposed sjekker om worldview-kartet inneholder
// en orden der én heis er Unconfirmed mens minst én annen er DeleteProposed.
// Returnerer true og en beskrivelse av konflikten hvis det er tilfellet.
func detectUnconfirmedVsDeleteProposed(worldviews map[string]wv.Worldview) (bool, string) {
	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			hasUnconfirmed := false
			hasDeleteProposed := false
			var unconfirmedID, deleteProposedID string

			for id, wview := range worldviews {
				if wview.Dead {
					continue
				}
				state := wview.HallOrders[f][d].SyncState
				if state == wv.Unconfirmed {
					hasUnconfirmed = true
					unconfirmedID = id
				}
				if state == wv.DeleteProposed {
					hasDeleteProposed = true
					deleteProposedID = id
				}
			}

			if hasUnconfirmed && hasDeleteProposed {
				return true, fmt.Sprintf(
					"Deadlock oppdaget: etasje=%d retning=%d — %s=Unconfirmed, %s=DeleteProposed",
					f, d, unconfirmedID, deleteProposedID,
				)
			}
		}
	}
	return false, ""
}

// TestUnconfirmedVsDeleteProposedDeadlock reproduserer scenarioet:
// A=Unconfirmed, B=DeleteProposed, C=DeleteProposed
// og verifiserer at sync ikke løser deadlocken.
func TestUnconfirmedVsDeleteProposedDeadlock(t *testing.T) {
	const floor = 1
	const dir = 0 // HallUp

	// Sett opp starttilstanden: A=Unconfirmed, B=DeleteProposed, C=DeleteProposed
	makeHallOrders := func(state wv.OrderSyncState) wv.HallOrders {
		var ho wv.HallOrders
		ho[floor][dir] = makeOrder(state)
		return ho
	}

	worldviews := map[string]wv.Worldview{
		"A": makeWorldview("A", makeHallOrders(wv.Unconfirmed)),
		"B": makeWorldview("B", makeHallOrders(wv.DeleteProposed)),
		"C": makeWorldview("C", makeHallOrders(wv.DeleteProposed)),
	}

	// Verifiser at vi faktisk har deadlock-tilstanden
	detected, msg := detectUnconfirmedVsDeleteProposed(worldviews)
	if !detected {
		t.Fatal("Forventet å detektere Unconfirmed vs DeleteProposed, men ble ikke funnet")
	}
	t.Logf("Deadlock bekreftet: %s", msg)

	// Kjør syncHallOrders for alle tre heisene og se om deadlocken løser seg
	lightsOnCh := make(chan [2]int, 10)
	lightsOffCh := make(chan [2]int, 10)

	const rounds = 5
	for round := 0; round < rounds; round++ {
		for id := range worldviews {
			synced := syncHallOrders(worldviews, id, lightsOnCh, lightsOffCh)
			updated := worldviews[id]
			updated.HallOrders = synced
			worldviews[id] = updated
		}
	}

	// Sjekk tilstandene etter synkronisering
	stateA := worldviews["A"].HallOrders[floor][dir].SyncState
	stateB := worldviews["B"].HallOrders[floor][dir].SyncState
	stateC := worldviews["C"].HallOrders[floor][dir].SyncState

	t.Logf("Tilstand etter %d synk-runder: A=%v, B=%v, C=%v", rounds, syncStateName(stateA), syncStateName(stateB), syncStateName(stateC))

	// Deadlocken skal fortsatt eksistere etter synkronisering
	stillDeadlocked, stillMsg := detectUnconfirmedVsDeleteProposed(worldviews)
	if !stillDeadlocked {
		t.Logf("OBS: Deadlocken løste seg av seg selv (uventet). Ny tilstand: A=%v, B=%v, C=%v",
			syncStateName(stateA), syncStateName(stateB), syncStateName(stateC))
	} else {
		t.Logf("Deadlock vedvarer etter %d runder: %s", rounds, stillMsg)
	}
}

// TestHowDeadlockArises viser steg-for-steg hvordan deadlocken oppstår:
// Alle starter Confirmed → B og C betjener etasjen → A trykker knapp igjen
func TestHowDeadlockArises(t *testing.T) {
	const floor = 2
	const dir = 1 // HallDown

	makeHallOrders := func(state wv.OrderSyncState) wv.HallOrders {
		var ho wv.HallOrders
		ho[floor][dir] = makeOrder(state)
		return ho
	}

	lightsOnCh := make(chan [2]int, 10)
	lightsOffCh := make(chan [2]int, 10)

	// Steg 1: Alle starter Confirmed (ordren er bekreftet)
	worldviews := map[string]wv.Worldview{
		"A": makeWorldview("A", makeHallOrders(wv.Confirmed)),
		"B": makeWorldview("B", makeHallOrders(wv.Confirmed)),
		"C": makeWorldview("C", makeHallOrders(wv.Confirmed)),
	}
	t.Log("Steg 1 — Alle Confirmed")

	// Steg 2: B betjener etasjen, setter DeleteProposed
	{
		updated := worldviews["B"]
		updated.HallOrders[floor][dir] = makeOrder(wv.DeleteProposed)
		worldviews["B"] = updated
		t.Log("Steg 2 — B setter DeleteProposed (betjener etasjen)")
	}

	// Steg 3: C synkroniserer og følger B til DeleteProposed
	{
		synced := syncHallOrders(worldviews, "C", lightsOnCh, lightsOffCh)
		updated := worldviews["C"]
		updated.HallOrders = synced
		worldviews["C"] = updated
		stateC := worldviews["C"].HallOrders[floor][dir].SyncState
		t.Logf("Steg 3 — C etter sync: %s", syncStateName(stateC))
	}

	// Steg 4: A opplever et nytt knappetrykk — addNewHallOrder overskriver til Unconfirmed
	{
		updated := worldviews["A"]
		updated.HallOrders[floor][dir] = makeOrder(wv.Unconfirmed) // simulerer addNewHallOrder
		worldviews["A"] = updated
		t.Log("Steg 4 — A trykker knapp igjen → Unconfirmed (overskrives fra Confirmed)")
	}

	stateA := worldviews["A"].HallOrders[floor][dir].SyncState
	stateB := worldviews["B"].HallOrders[floor][dir].SyncState
	stateC := worldviews["C"].HallOrders[floor][dir].SyncState
	t.Logf("Etter steg 4: A=%s, B=%s, C=%s",
		syncStateName(stateA), syncStateName(stateB), syncStateName(stateC))

	// Sjekk at vi nå er i deadlock
	detected, msg := detectUnconfirmedVsDeleteProposed(worldviews)
	if !detected {
		t.Fatal("Forventet deadlock etter steg 4, men ble ikke detektert")
	}
	t.Logf("Deadlock bekreftet: %s", msg)

	// Steg 5: Kjør synk og vis at ingen av dem kommer seg videre
	for i := 0; i < 3; i++ {
		for id := range worldviews {
			synced := syncHallOrders(worldviews, id, lightsOnCh, lightsOffCh)
			updated := worldviews[id]
			updated.HallOrders = synced
			worldviews[id] = updated
		}
	}

	stateA = worldviews["A"].HallOrders[floor][dir].SyncState
	stateB = worldviews["B"].HallOrders[floor][dir].SyncState
	stateC = worldviews["C"].HallOrders[floor][dir].SyncState
	t.Logf("Steg 5 — Etter 3 synk-runder: A=%s, B=%s, C=%s",
		syncStateName(stateA), syncStateName(stateB), syncStateName(stateC))

	if _, msg := detectUnconfirmedVsDeleteProposed(worldviews); msg != "" {
		t.Logf("Deadlock vedvarer: %s", msg)
	}
}
