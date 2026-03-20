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

// syncHallOrders synchronizes hall orders against all known peers and returns the updated state.
func syncHallOrders(
	latestWorldviews map[string]wv.Worldview,
	myID string,
) wv.HallOrders {
	myHallOrders := latestWorldviews[myID].HallOrders

	// Step 0: Propagate peerDied. If a live peer has {Unconfirmed, peerDied} and we have
	// {Confirmed, _}, we should follow it downward. The owner of the order has marked itself
	// as dead/error, and that information must be propagated even if we did not receive the
	// original broadcast from the failing elevator.
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

	// Step 1: Follow peers that are one step ahead, skipping only Dead peers and ourselves.
	// PeerDied orders, whether from Step 0 or worldview, must NOT be promoted here.
	// Only Step 2 consensus may advance them, with OwnerID clearing.
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
				} else if secondToNextOrderState(myCurrentOrder.SyncState) == peerCurrentOrder.SyncState && myCurrentOrder.OwnerID != wv.PeerDied {
					myHallOrders[f][d].SyncState = peerCurrentOrder.SyncState
				}
			}
		}
	}

	// Step 2: Consensus check. Advance the state if everyone agrees.
	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			myOrder := myHallOrders[f][d]

			switch myOrder.SyncState {

			case wv.Unconfirmed:
				// Require that we have already broadcast the Unconfirmed state before allowing consensus.
				// This prevents a Confirmed -> Unconfirmed downgrade (peerDied via Step 0) from jumping
				// directly back to Confirmed in the same round without first informing the others.
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

	// Step 3: OwnerID conflict. If two live peers disagree on who owns a Confirmed order,
	// choose the deterministic winner with the smallest OwnerID.
	// Both elevators converge independently to the same owner after one sync round.
	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
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

	// Step 4: Normalize. None orders should never have an owner.
	// Also ensure confirmed orders never have a dead/error/disconnected owner.
	for f := 0; f < wv.NumFloors; f++ {
		for d := 0; d < wv.Directions; d++ {
			order := myHallOrders[f][d]

			if order.SyncState == wv.None {
				myHallOrders[f][d].OwnerID = wv.NoOwner
			} else if order.SyncState == wv.Confirmed && order.OwnerID != wv.NoOwner && order.OwnerID != wv.PeerDied {
				// Check if the owner is alive
				if ownerWv, exists := latestWorldviews[order.OwnerID]; exists {
					if ownerWv.Dead || ownerWv.ErrorState {
						// Owner is dead/error - downgrade to unconfirmed with PeerDied marker
						myHallOrders[f][d].SyncState = wv.Unconfirmed
						myHallOrders[f][d].OwnerID = wv.PeerDied
					}
				} else {
					// Owner doesn't exist in worldviews - downgrade it
					myHallOrders[f][d].SyncState = wv.Unconfirmed
					myHallOrders[f][d].OwnerID = wv.PeerDied
				}
			}
		}
	}

	return myHallOrders
}

func GoroutineSync(
	myID string,
	syncedHallOrdersCh chan wv.HallOrders,
	worldviewsForSyncCh <-chan map[string]wv.Worldview,
) {
	for {
		latestWorldviews := <-worldviewsForSyncCh
		syncedHallOrders := syncHallOrders(latestWorldviews, myID)
		select {
		case syncedHallOrdersCh <- syncedHallOrders:
		default:
			select {
			case <-syncedHallOrdersCh:
			default:
			}
			syncedHallOrdersCh <- syncedHallOrders
		}
	}
}
