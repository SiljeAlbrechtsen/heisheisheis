package synchronization


// ____________________________________________________________________________________________________________
// ---------------- CHANNELS-----------------------------------------------------------------------------------
// ____________________________________________________________________________________________________________

// Inn channel: worldview-map

// Ut channel: sende ut hallOrders

//____________________________________________________________________________________________________________
//----------------  FUNKSJONER FOR Å HÅNDTERE WORLDVIEW ------------------------------------------------------
//____________________________________________________________________________________________________________

func nextOrderState(currentSyncState orderSyncState) orderSyncState {
	switch currentSyncState {
	case None:
		return Unconfirmed
	case Unconfirmed:
		return Confirmed
	case Confirmed:
		return DeleteProposed
	case DeleteProposed:
		return None
	default:
		return None
	}
}

// Trigges når vi får inn nye worldviews
func syncHallOrders(latestWorldviews map[int]Worldview) HallOrders {
	var myHallOrders HallOrders

	for _, peer := latestWorldviews {
		myHallOrders = peer.hallorders
		break
	}

	// Itererer gjennom hele map. TODO: itererer også gjennom seg selv
	for _, peer := range latestWorldviews {
		//Iterere gjennom hallOrdersene
		for f := 0; f < NumFloors; f++ {
			for d := 0; d < Directions; d++ {
				
				myCurrentOrder := myHallOrders[f][d]
				peerCurrentOrder := peer.hallOrders[f][d]

				if  myCurrentOrder == peerCurrentOrder {
					continue

				// TODO: slå sammen?
				// Hvis peer er på next order skal jeg også på next order
				} else if nextOrderState(myCurrentOrder.syncState) == peerCurrentOrder.orderSyncState {
					myHallOrders[f][d] = peerCurrentOrder

				} else if myCurrentOrder == nextOrderState(peerCurrentOrder) && peerCurrentOrder.ownerID == peerDied{
					myHallOrders[f][d] = peerCurrentOrder 
				}
			}
		}
	}
	return myHallOrders
	// TODO: må brukes for å oppdatere worldview. 
}