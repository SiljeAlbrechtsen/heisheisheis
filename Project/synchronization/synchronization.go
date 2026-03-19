package synchronization

import (
	wv "Project/worldview"
)

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

func secondToNextOrderState(currentSyncState wv.OrderSyncState) wv.OrderSyncState {
	switch currentSyncState {

	case wv.None:
		return wv.Confirmed

	case wv.Unconfirmed:
		return wv.DeleteProposed

	default:
		return wv.None
	}
}

// Trigges når vi får inn nye worldviews. Synkroniserer hall orders og sender på channel når lys skal skrus på/av.
func syncHallOrders(
	latestWorldviews map[string]wv.Worldview,
	myID string,
) wv.HallOrders {
	myHallOrders := latestWorldviews[myID].HallOrders

	// Steg 0: Propager peerDied — hvis en alive peer har {Unconfirmed, peerDied} og vi har
	// {Confirmed, _}, skal vi følge ned. Eieren av ordren har markert seg som dead/error,
	// og denne infoen må spres selv om vi ikke fikk originalbroadcastet fra den feilende heisen.
	for _, peer := range latestWorldviews {
		if peer.Dead {
			continue
		}
		for f := 0; f < wv.NumFloors; f++ {
			for d := 0; d < wv.Directions; d++ {
				peerOrder := peer.HallOrders[f][d]
				if peerOrder.SyncState == wv.Unconfirmed &&
					peerOrder.OwnerID == wv.PeerDied &&
					myHallOrders[f][d].SyncState == wv.Confirmed {
					myHallOrders[f][d] = wv.Order{SyncState: wv.Unconfirmed, OwnerID: wv.PeerDied}
				}
			}
		}
	}

	// Steg 1: Følg peers som er ett steg foran (hopp kun over Dead og seg selv)
	// PeerDied-ordrer (fra Steg 0 eller worldview) skal IKKE promoteres her —
	// kun Steg 2 konsensus kan avansere dem (med OwnerID-clearing).
	for _, peer := range latestWorldviews {
		if peer.Dead || peer.IdElevator == myID {
			continue
		}
		for f := 0; f < wv.NumFloors; f++ {
			for d := 0; d < wv.Directions; d++ {
				myCurrentOrder := myHallOrders[f][d]
				peerCurrentOrder := peer.HallOrders[f][d]

				if myCurrentOrder == peerCurrentOrder {
					continue

				} else if nextOrderState(myCurrentOrder.SyncState) == peerCurrentOrder.SyncState && myCurrentOrder.OwnerID != wv.PeerDied {
					myHallOrders[f][d].SyncState = peerCurrentOrder.SyncState
				}
				if secondToNextOrderState(myCurrentOrder.SyncState) == peerCurrentOrder.SyncState && myCurrentOrder.OwnerID != wv.PeerDied {
					myHallOrders[f][d].SyncState = peerCurrentOrder.SyncState
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
				// Krev at vi allerede har broadcast Unconfirmed-state før vi tillater konsensus.
				// Hindrer at Confirmed→Unconfirmed-degradering (peerDied via Steg 0) hopper
				// direkte til Confirmed i samme runde uten å ha fortalt andre om det.
				if latestWorldviews[myID].HallOrders[f][d].SyncState != wv.Unconfirmed {
					break
				}
				allAgree := true
				for _, peer := range latestWorldviews {
					if peer.Dead {
						continue
					}
					peerState := peer.HallOrders[f][d].SyncState
					if peerState != wv.Unconfirmed && peerState != wv.Confirmed {
						allAgree = false
						break
					}
				}
				if allAgree {
					myHallOrders[f][d].SyncState = wv.Confirmed
					if myHallOrders[f][d].OwnerID == wv.PeerDied {
						myHallOrders[f][d].OwnerID = wv.NoOwner
					}
				}

			case wv.DeleteProposed:
				allAgree := true
				for _, peer := range latestWorldviews {
					if peer.Dead {
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
				}
			}
		}
	}

	// Steg 3: Normaliser — None-ordrer skal aldri ha owner
	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			if myHallOrders[f][d].SyncState == wv.None {
				myHallOrders[f][d].OwnerID = wv.NoOwner
			}
		}
	}

	return myHallOrders
}

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
