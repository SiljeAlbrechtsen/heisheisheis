package synchronization

import (
	"fmt"

	wv "Project/worldview"
)

func syncStateName(s wv.OrderSyncState) string {
	switch s {
	case wv.None:
		return "None"
	case wv.Unconfirmed:
		return "Unconfirmed"
	case wv.Confirmed:
		return "Confirmed"
	case wv.DeleteProposed:
		return "DeleteProposed"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

func dirName(d int) string {
	if d == 0 {
		return "Up"
	}
	return "Down"
}

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

func canAdvanceUnconfirmedByConsensus(latestWorldviews map[string]wv.Worldview, myID string, floor, dir int) bool {
	allAgree := true
	seenByOtherPeer := false

	for id, peer := range latestWorldviews {
		if peer.Dead {
			continue
		}
		peerState := peer.HallOrders[floor][dir].SyncState
		if peerState != wv.Unconfirmed && peerState != wv.Confirmed {
			allAgree = false
			break
		}
		if id != myID && peerState == wv.Unconfirmed {
			seenByOtherPeer = true
		}
	}

	return allAgree && seenByOtherPeer
}

func canAdvanceDeleteByConsensus(latestWorldviews map[string]wv.Worldview, myID string, floor, dir int) bool {
	allAgree := true
	seenByOtherPeer := false

	for id, peer := range latestWorldviews {
		if peer.Dead {
			continue
		}
		peerState := peer.HallOrders[floor][dir].SyncState
		if peerState != wv.DeleteProposed && peerState != wv.None {
			allAgree = false
			break
		}
		if id != myID && peerState == wv.DeleteProposed {
			seenByOtherPeer = true
		}
	}

	return allAgree && seenByOtherPeer
}

// Trigges når vi får inn nye worldviews. Synkroniserer hall orders og sender på channel når lys skal skrus på/av.
func syncHallOrders(
	latestWorldviews map[string]wv.Worldview,
	myID string,
) wv.HallOrders {
	myHallOrders := latestWorldviews[myID].HallOrders

	peerList := ""
	for id, peer := range latestWorldviews {
		if peer.ErrorState {
			peerList += id + "(dead) "
		} else {
			peerList += id + " "
		}
	}
	_ = peerList

	// Steg 1: Følg peers som er nøyaktig ett steg foran. Ingen hopp over mellomstater.
	for _, peer := range latestWorldviews {
		if peer.Dead {
			continue
		}
		for f := 0; f < wv.NumFloors; f++ {
			for d := 0; d < wv.Directions; d++ {
				myCurrentOrder := myHallOrders[f][d]
				peerCurrentOrder := peer.HallOrders[f][d]

				if myCurrentOrder == peerCurrentOrder {
					continue
				}

				if nextOrderState(myCurrentOrder.SyncState) == peerCurrentOrder.SyncState {
					myHallOrders[f][d].SyncState = peerCurrentOrder.SyncState
					if peerCurrentOrder.SyncState == wv.None {
						myHallOrders[f][d].OwnerID = wv.NoOwner
					}
				}
			}
		}
	}

	// Steg 2: Konsensus krever at alle levende peers er i tillatt del av syklusen
	// og at minst én annen peer faktisk har observert forslaget.
	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			myOrder := myHallOrders[f][d]

			switch myOrder.SyncState {
			case wv.Unconfirmed:
				if canAdvanceUnconfirmedByConsensus(latestWorldviews, myID, f, d) {
					myHallOrders[f][d].SyncState = wv.Confirmed
					if myHallOrders[f][d].OwnerID == wv.PeerDied {
						myHallOrders[f][d].OwnerID = wv.NoOwner
					}
				}

			case wv.DeleteProposed:
				if canAdvanceDeleteByConsensus(latestWorldviews, myID, f, d) {
					myHallOrders[f][d] = wv.Order{SyncState: wv.None, OwnerID: wv.NoOwner}
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
	myID string,
	syncToWorldviewCh chan wv.HallOrders,
	worldviewToSyncCh <-chan map[string]wv.Worldview,
) {
	for {
		latestWorldviews := <-worldviewToSyncCh
		syncedHallOrders := syncHallOrders(latestWorldviews, myID)
		select {
		case syncToWorldviewCh <- syncedHallOrders:
		default:
			select {
			case <-syncToWorldviewCh:
			default:
			}
			syncToWorldviewCh <- syncedHallOrders
		}
	}
}
