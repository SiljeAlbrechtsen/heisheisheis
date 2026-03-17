package synchronization

import (
	wv "Project/worldview"
	"fmt"
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

// Trigges når vi får inn nye worldviews. Synkroniserer hall orders og sender på channel når lys skal skrus på/av.
func syncHallOrders(
	latestWorldviews map[string]wv.Worldview,
	myID string,
	lightsOnCh chan<- [2]int,
	lightsOffCh chan<- [2]int,
) wv.HallOrders {
	myHallOrders := latestWorldviews[myID].HallOrders

	// Logg hvem vi synkroniserer med
	peerList := ""
	for id, peer := range latestWorldviews {
		if peer.ErrorState {
			peerList += id + "(dead) "
		} else {
			peerList += id + " "
		}
	}
	fmt.Printf("[Sync] Starter synk for %s | peers: %s\n", myID, peerList)

	// Steg 1: Følg peers som er ett steg foran
	for _, peer := range latestWorldviews {
		if peer.ErrorState {
			continue
		}
		for f := 0; f < wv.NumFloors; f++ {
			for d := 0; d < wv.Directions; d++ {
				myCurrentOrder := myHallOrders[f][d]
				peerCurrentOrder := peer.HallOrders[f][d]

				if myCurrentOrder == peerCurrentOrder {
					continue

				} else if nextOrderState(myCurrentOrder.SyncState) == peerCurrentOrder.SyncState {
					fmt.Printf("[Sync][Steg1] Følger %s: floor=%d dir=%s %s->%s\n",
						peer.IdElevator, f, dirName(d),
						syncStateName(myCurrentOrder.SyncState),
						syncStateName(peerCurrentOrder.SyncState))
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
						fmt.Printf("[Sync][Steg2] Ikke konsensus Unconfirmed: floor=%d dir=%s peer=%s er %s\n",
							f, dirName(d), peer.IdElevator,
							syncStateName(peer.HallOrders[f][d].SyncState))
						allAgree = false
						break
					}
				}
				if allAgree {
					fmt.Printf("[Sync][Steg2] Konsensus! Unconfirmed->Confirmed floor=%d dir=%s\n", f, dirName(d))
					myHallOrders[f][d].SyncState = wv.Confirmed
					if myHallOrders[f][d].OwnerID == wv.PeerDied {
						myHallOrders[f][d].OwnerID = wv.NoOwner
					}
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
						fmt.Printf("[Sync][Steg2] DeleteProposed blokkert: floor=%d dir=%s peer=%s er %s\n",
							f, dirName(d), peer.IdElevator, syncStateName(peerState))
						break
					}
				}
				if allAgree {
					fmt.Printf("[Sync][Steg2] Konsensus! DeleteProposed->None floor=%d dir=%s\n", f, dirName(d))
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
	myID string,
	syncToWorldviewCh chan<- wv.HallOrders,
	worldviewToSyncCh <-chan map[string]wv.Worldview,
	lightsOnCh chan<- [2]int,
	lightsOffCh chan<- [2]int,
) {
	for {
		latestWorldviews := <-worldviewToSyncCh
		fmt.Printf("[Sync] Mottok worldview-oppdatering (%d peers)\n", len(latestWorldviews))
		syncedHallOrders := syncHallOrders(latestWorldviews, myID, lightsOnCh, lightsOffCh)
		fmt.Printf("[Sync] Sender synkede hallorders tilbake til worldview\n")
		syncToWorldviewCh <- syncedHallOrders
	}
}
