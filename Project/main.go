package main

// Map for å lagre siste heartbeats til alle nye heiser 
lastWorldview := make(map[string]Worldview)

func addNewPeerToWorldview(p peers.PeerUpdate, hbMap map[string]Worldview){
	// Sjekke om ny heis finnes i map, hvis ikke legg til default worldview
	// sjekke om den ligger i map
	// Hvis ikke opprett ny nøkkel i map med default verdi

	for  

}

func updatePeerWorldview(p peers.PeerUpdate, hbMap map[string]Worldview){
}