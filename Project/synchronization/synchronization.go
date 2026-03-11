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

// Følger peers som er ett steg foran, og håndterer peer died.
func syncHallOrders(latestWorldviews map[int]wv.Worldview, myID int) wv.HallOrders {
	myHallOrders := latestWorldviews[myID].hallOrders

	for _, peer := range latestWorldviews {
		for f := 0; f < wv.NumFloors; f++ {
			for d := 0; d < wv.Directions; d++ {
				myCurrentOrder := myHallOrders[f][d]
				peerCurrentOrder := peer.hallOrders[f][d]

				if myCurrentOrder == peerCurrentOrder {
					continue

				// Hvis peer er på next order skal jeg også på next order
				} else if nextOrderState(myCurrentOrder.syncState) == peerCurrentOrder.syncState {
					myHallOrders[f][d] = peerCurrentOrder

				// Hvis peer har dødd og er ett steg bak skal vi gå tilbake
				} else if myCurrentOrder.syncState == nextOrderState(peerCurrentOrder.syncState) && peerCurrentOrder.ownerID == wv.PeerDied {
					myHallOrders[f][d] = peerCurrentOrder
				}
			}
		}
	}
	return myHallOrders
}

// Avanserer state når alle peers er enige, og sender på lys-channels.
func applyConsensus(
	myHallOrders     wv.HallOrders,
	latestWorldviews map[int]wv.Worldview,
	lightsOnCh       chan<- [2]int,
	lightsOffCh      chan<- [2]int,
) wv.HallOrders {
	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			switch myHallOrders[f][d].syncState {

			case wv.Unconfirmed:
				allAgree := true
				for _, peer := range latestWorldviews {
					// Hvis en peer har syncstate = none, så er de ikke enige
					if peer.hallOrders[f][d].syncState == wv.None {
						allAgree = false
						break
					}
				}
				if allAgree {
					myHallOrders[f][d] = wv.Order{syncState: wv.Confirmed, ownerID: wv.NoOwner}
					lightsOnCh <- [2]int{f, d}
				}

			case wv.DeleteProposed:
				allAgree := true
				for _, peer := range latestWorldviews {
					peerState := peer.hallOrders[f][d].syncState
					if peerState == wv.Confirmed  {
						allAgree = false
						break
					}
				}
				if allAgree {
					myHallOrders[f][d] = wv.Order{syncState: wv.None, ownerID: wv.NoOwner}
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

func goRoutineSync(
	myID              int,
	syncToWorldviewCh chan<- wv.HallOrders,
	worldviewToSyncCh <-chan map[int]wv.Worldview,
	lightsOnCh        chan<- [2]int,
	lightsOffCh       chan<- [2]int,
) {
	for {
		latestWorldviews := <-worldviewToSyncCh
		synced := syncHallOrders(latestWorldviews, myID)
		final  := applyConsensus(synced, latestWorldviews, lightsOnCh, lightsOffCh)
		syncToWorldviewCh <- final
	}
}






