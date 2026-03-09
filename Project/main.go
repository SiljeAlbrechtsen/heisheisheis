package main

const (
	MyID    int
)



func main() {
	//__________________________________________________________
	//-------------------Channels-------------------------------
	//__________________________________________________________

	// Assignment


	// FSM

	// NETWORK

	// Kanal for å motta id til peers som er på nett
	peerUpdateCh := make(chan peers.PeerUpdate)
	//Disable/enable transmitter etter den har startet, kan brukes til å signalisere at vi er utilgjengelig
	peerTxEnable := make(chan bool)  // Unødvendig??
	//Sende vår worldview
	worldviewTx := make(chan wordlview.WorldView)
	//Motta andre worldviews
	worldviewRx := make(chan wordlview.WorldView)

	// Synchronization
	syncedHallOrdersCh := make(chan worldview.HallOrders)

	// Worldview
	latesWorldviewsCh := make(chan worldView.map[string]Worldview)

	//___________________________________________________________
	//------------------ go rutines -----------------------------
	//___________________________________________________________

	/*
	- Transmitte wv
	- Recieve wv
		- addPeerToMap
		- updateLatesWordviews
	- synce
	- assigne
	- fsm
	*/

}














}

