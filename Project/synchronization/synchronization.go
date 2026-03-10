package synchronization

import (
	wv "Project/worldview"
)


// ____________________________________________________________________________________________________________
// ---------------- CHANNELS-----------------------------------------------------------------------------------
// ____________________________________________________________________________________________________________

// Inn channel: worldview-map            worldviewToSyncCh

// Ut channel: sende ut hallOrders.      syncToWorldviewCh

//____________________________________________________________________________________________________________
//----------------  FUNKSJONER FOR Å HÅNDTERE WORLDVIEW ------------------------------------------------------
//____________________________________________________________________________________________________________

func nextOrderState(currentSyncState wv.orderSyncState) wv.orderSyncState {
	switch currentSyncState {

	case wv.None:
		return wv.Unconfirmed

	case wv.Unconfirmed:
		return wv.Confirmed

	case wv.Confirmed:
		return wv.DeleteProposed

	case wv.DeleteProposed:
		return wv.None

	default:
		return wv.None
	}
}

// Trigges når vi får inn nye worldviews
func syncHallOrders(latestWorldviews map[int]wv.Worldview) wv.hallOrders {
	var myHallOrders wv.hallOrders

	for _, peer := range latestWorldviews {
		myHallOrders = peer.hallOrders
		break
	}

	// Itererer gjennom hele map. TODO: itererer også gjennom seg selv
	for _, peer := range latestWorldviews {
		//Iterere gjennom hallOrdersene
		for f := 0; f < wv.NumFloors; f++ {
			for d := 0; d < wv.Directions; d++ {
				
				myCurrentOrder := myHallOrders[f][d]
				peerCurrentOrder := peer.hallOrders[f][d]

				if  myCurrentOrder == peerCurrentOrder {
					continue

				// TODO: slå sammen?
				// Hvis peer er på next order skal jeg også på next order
				} else if nextOrderState(myCurrentOrder.syncState) == peerCurrentOrder.orderSyncState {
					myHallOrders[f][d] = peerCurrentOrder

					// Hvis vi er på confirmed, peer er på unconfirmed, men har dødd skal vi også gå til unconfirmed. 
				} else if myCurrentOrder.syncState == nextOrderState(peerCurrentOrder.syncState) && peerCurrentOrder.ownerID == wv.PeerDied{
					myHallOrders[f][d] = peerCurrentOrder 
				}
			}
		}
	}
	return myHallOrders
}


// _______________________________________________________
// ---------------GO ROUTINE MED CHANNELS-----------------
// _______________________________________________________

func goRoutineSync(
	//latestWorldviews map[int]Worldview, 
	syncToWorldviewCh chan<- wv.hallOrders,
	worldviewToSyncCh <-chan map[int]wv.Worldview,
	) {
	for {
		latestWorldviews := <-worldviewToSyncCh
		syncedHallOrders := syncHallOrders(latestWorldviews)
		syncToWorldviewCh <- syncedHallOrders
	}
}






/*
Finne ut hvor disse funk skal stå. Sync?
-Sjekke om alle har en ordre som er på proposedDeleted -> Da skal den settes til No Order og No Owner og skru av lys via channel til FSM
-Må ha en funksjon som sjekker om alle har unconfirmed order -> Da skal den gjøre om til confirmed. Lys skal skru på via channel til FSM

Grunnen til at de har med bool er slik at vi da kan sende channel fra worldview til lys om at de må skrues på. 
Evt derfor ha de i ulike
Klarer ikke bestemme meg for om det gir mer cohesion eller mindre
*/

// Denne itererer gjennom alle hall orders og sjekker. Klarer ikke å tenke om det er nødvendig atm
func confirmIfAllAgree(worldviewsMap map[int]Worldview, myID int) (HallOrders, bool) {
	myOrders := worldviewsMap[myID].hallOrders
	changed := false

	// Itererer gjennom hall orders til vår heis
    for f := 0; f < NumFloors; f++ {
        for d := 0; d < Directions; d++ {
            order := myOrders[f][d]

			// Sjekker om vi har noen orders som er unconfirmed
            if order.syncState != Unconfirmed {
                continue
            }

			// Antar først at alle er enige. Hvis noen andre har ordersyncstate til None så settes den til false
			// Må den gjelde noen andre enn false?
            allAgree := true
            for _, peer := range worldviewsMap {
                peerState := peer.hallOrders[f][d].syncState
                if peerState == None {
                    allAgree = false
                    break
                }
            }

            // Hvis alle er enige, så oppdater staten 
            if allAgree {
                myOrders[f][d] = Order{
                    syncState: Confirmed,
					// Setter ownerID etter den har vært i assigned. Evt trenger vi den? Ja tror det. Hvor skal den være?
                    ownerID:   noOwner,
                }
            }
        }
    }
    return myOrders, changed
}


func deleteIfAllAgree(worldviewsMap map[int]Worldview, myID int) (HallOrders, bool) {
	myOrders := worldviewsMap[myID].hallOrders
	changed := false

    for f := 0; f < NumFloors; f++ {
        for d := 0; d < Directions; d++ {
            if myOrders[f][d].syncState != DeleteProposed {
                continue
            }

            allAgree := true
            for _, peer := range worldviewsMap {
                peerState := peer.hallOrders[f][d].syncState
                if peerState != DeleteProposed && peerState != None {
                    allAgree = false
                    break
                }
            }


            if allAgree {
                myOrders[f][d] = Order{
                    syncState: None,
                    ownerID:   noOwner,
                }
                changed = true
            }
        }
    }
    return myOrders, changed
}


// Jeg føler at disse overlapper hverandre endel og kan skrives sammen for det er samme ansvarsområde
// Claude foreslo denne funksjonen, men jeg klarer ikke helt å se den sammenhengen akkurat nå. Head == cooked

func updateHallOrders(worldviewsMap map[int]Worldview, myID int) (HallOrders, bool) {
    myOrders := worldviewsMap[myID].hallOrders
    changed := false

    for f := 0; f < NumFloors; f++ {
        for d := 0; d < Directions; d++ {
            myOrder := myOrders[f][d]

            allAgree := true
            anyAhead := false

            for _, peer := range worldviewsMap {
                peerOrder := peer.hallOrders[f][d]

                // Peer er ett steg foran → følg etter
                if nextOrderState(myOrder.syncState) == peerOrder.syncState {
                    anyAhead = true
                    myOrders[f][d] = peerOrder
                    changed = true
                }

                // Konsensussjekk
                if peerOrder.syncState == None {
                    allAgree = false
                }
            }

            // Hvis alle er enige og ingen var foran
            if allAgree && !anyAhead && myOrder.syncState == Unconfirmed {
                myOrders[f][d].syncState = Confirmed
                changed = true
            }
        }
    }
    return myOrders, changed
}