package synchronization

import (
	elev "Project/elevator"
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

func allAliveAgree(worldviews map[string]wv.Worldview, f, d int, accepted ...wv.OrderSyncState) bool {
	for _, peer := range worldviews {
		if peer.Dead {
			continue
		}
		peerState := peer.HallOrders[f][d].SyncState
		ok := false
		for _, s := range accepted {
			if peerState == s {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
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

// syncHallOrders synkroniserer hall orders mot alle kjente peers og returnerer oppdatert tilstand.
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
		for f := 0; f < elev.N_FLOORS; f++ {
			for d := 0; d < elev.N_DIRECTIONS; d++ {
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
		for f := 0; f < elev.N_FLOORS; f++ {
			for d := 0; d < elev.N_DIRECTIONS; d++ {
				myCurrentOrder := myHallOrders[f][d]
				peerCurrentOrder := peer.HallOrders[f][d]

				if myCurrentOrder == peerCurrentOrder {
					continue

				} else if nextOrderState(myCurrentOrder.SyncState) == peerCurrentOrder.SyncState && myCurrentOrder.OwnerID != wv.PeerDied {
					myHallOrders[f][d].SyncState = peerCurrentOrder.SyncState
				} else if secondToNextOrderState(myCurrentOrder.SyncState) == peerCurrentOrder.SyncState && myCurrentOrder.OwnerID != wv.PeerDied {
					myHallOrders[f][d].SyncState = peerCurrentOrder.SyncState
				}
			}
		}
	}

	// Steg 2: Konsensussjekk — avanser state hvis alle er enige
	for f := 0; f < elev.N_FLOORS; f++ {
		for d := 0; d < elev.N_DIRECTIONS; d++ {
			myOrder := myHallOrders[f][d]

			switch myOrder.SyncState {

			case wv.Unconfirmed:
				// Krev at vi allerede har broadcast Unconfirmed-state før vi tillater konsensus.
				// Hindrer at Confirmed→Unconfirmed-degradering (peerDied via Steg 0) hopper
				// direkte til Confirmed i samme runde uten å ha fortalt andre om det.
				if latestWorldviews[myID].HallOrders[f][d].SyncState != wv.Unconfirmed {
					break
				}
				if allAliveAgree(latestWorldviews, f, d, wv.Unconfirmed, wv.Confirmed) {
					myHallOrders[f][d].SyncState = wv.Confirmed
					if myHallOrders[f][d].OwnerID == wv.PeerDied {
						myHallOrders[f][d].OwnerID = wv.NoOwner
					}
				}

			case wv.DeleteProposed:
				if allAliveAgree(latestWorldviews, f, d, wv.DeleteProposed, wv.None) {
					myHallOrders[f][d] = wv.Order{SyncState: wv.None, OwnerID: wv.NoOwner}
				}
			}
		}
	}

	// Steg 3: OwnerID-konflikt — hvis to alive peers er uenige om hvem som eier en Confirmed ordre,
	// velg deterministisk vinner med minste OwnerID.
	// Begge heiser konvergerer uavhengig til samme eier etter én sync-runde.
	for f := 0; f < elev.N_FLOORS; f++ {
		for d := 0; d < elev.N_DIRECTIONS; d++ {
			myOrder := myHallOrders[f][d]
			if myOrder.SyncState != wv.Confirmed || myOrder.OwnerID == wv.NoOwner || myOrder.OwnerID == wv.PeerDied {
				continue
			}
			for _, peer := range latestWorldviews {
				if peer.Dead || peer.IdElevator == myID {
					continue
				}
				peerOrder := peer.HallOrders[f][d]
				if peerOrder.SyncState == wv.Confirmed &&
					peerOrder.OwnerID != wv.NoOwner &&
					peerOrder.OwnerID != wv.PeerDied &&
					peerOrder.OwnerID != myOrder.OwnerID {
					if peerOrder.OwnerID < myOrder.OwnerID {
						myHallOrders[f][d].OwnerID = peerOrder.OwnerID
					}
					break
				}
			}
		}
	}

	// Steg 4: Normaliser — None-ordrer skal aldri ha owner
	for f := 0; f < elev.N_FLOORS; f++ {
		for d := 0; d < elev.N_DIRECTIONS; d++ {
			if myHallOrders[f][d].SyncState == wv.None {
				myHallOrders[f][d].OwnerID = wv.NoOwner
			}
		}
	}

	return myHallOrders
}

func GoroutineSync(
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
