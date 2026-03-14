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

func nextOrderState(currentSyncState wv.OrderSyncState) wv.OrderSyncState {
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

// Trigges når vi får inn nye worldviews. Synkroniserer hall orders og sender på channel når lys skal skrus på/av.
func syncHallOrders(
	latestWorldviews map[string]wv.Worldview,
	myID string,
	lightsOnCh  chan<- [2]int,
	lightsOffCh chan<- [2]int,
) wv.HallOrders {
	myHallOrders := latestWorldviews[myID].HallOrders

	// Steg 1: Følg peers som er ett steg foran
	for _, peer := range latestWorldviews {
		for f := 0; f < wv.NumFloors; f++ {
			for d := 0; d < wv.Directions; d++ {
				myCurrentOrder := myHallOrders[f][d]
				peerCurrentOrder := peer.HallOrders[f][d]

				if myCurrentOrder == peerCurrentOrder {
					continue

				// Hvis peer er på next order skal jeg også på next order
				} else if nextOrderState(myCurrentOrder.SyncState) == peerCurrentOrder.SyncState {
					myHallOrders[f][d] = peerCurrentOrder

				// Hvis vi er på confirmed, peer er på unconfirmed, men har dødd skal vi også gå til unconfirmed.
				} else if myCurrentOrder.SyncState == nextOrderState(peerCurrentOrder.SyncState) && peerCurrentOrder.OwnerID == wv.PeerDied {
					myHallOrders[f][d] = peerCurrentOrder
				}
			}
		}
	}

	// Steg 2: Konsensussjekk — avanser state hvis alle er enige
	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			myOrder := myHallOrders[f][d]

			switch myOrder.SyncState {

			case wv.Unconfirmed:
				allAgree := true
				for _, peer := range latestWorldviews {
					if peer.ErrorState {
						continue
					}
					if peer.HallOrders[f][d].SyncState != wv.Unconfirmed {
						allAgree = false
						break
					}
				}
				if allAgree {
					myHallOrders[f][d].SyncState = wv.Confirmed
					lightsOnCh <- [2]int{f, d}
				}

			case wv.DeleteProposed:
				allAgree := true
				for _, peer := range latestWorldviews {
					if peer.ErrorState {
						continue
					}
					peerState := peer.HallOrders[f][d].SyncState
					if peerState != wv.DeleteProposed && peerState != wv.None {
						allAgree = false
						break
					}
				}
				if allAgree {
					myHallOrders[f][d] = wv.Order{SyncState: wv.None, OwnerID: wv.NoOwner}
					lightsOffCh <- [2]int{f, d}
				}
			}
		}
	}

	return myHallOrders
}


// _______________________________________________________
// ---------------GO ROUTINE MED CHANNELS-----------------
// _______________________________________________________

func GoRoutineSync(
	myID              string,
	syncToWorldviewCh chan<- wv.HallOrders,
	worldviewToSyncCh <-chan map[string]wv.Worldview,
	lightsOnCh        chan<- [2]int,
	lightsOffCh       chan<- [2]int,
) {
	for {
		latestWorldviews := <-worldviewToSyncCh
		syncedHallOrders := syncHallOrders(latestWorldviews, myID, lightsOnCh, lightsOffCh)
		syncToWorldviewCh <- syncedHallOrders
	}
}






