package main

import (
	"Network/network/bcast"
	"Network/network/localip"
	"Network/network/peers"
	"flag"
	"fmt"
	"os"
	"time"
)

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//  will be received as zero-values.

// Mulig structsene kan defineres et annet sted
type ElevatorState struct {
	Floor    int
	Dir      bool
	DoorOpen bool
	Idle     bool
	Error    bool
}

type OrderStatus int

const (
	OrderUnassigned OrderStatus = iota
	OrderAssigned
	OrderServed
)

type OrderSyncState int

const (
	SyncUnconfirmed OrderSyncState = iota
	SyncConfirmed
	SyncDeleteProposed
)

type Order struct {
	Floor     int
	Direction bool
	Status    OrderStatus
	SyncState OrderSyncState
	OwnerID   int
}

type Heartbeat struct {
	SenderID      string
	ElevatorState ElevatorState // finnes noe lignene fra før?
	HallOrders    []Order
	CabOrders     []Order
}

func main() {

	//__________________________________________________________________
	//----------------  SETTER UNIK ID FOR DENNE NODEN -----------------
	//__________________________________________________________________

	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	// ... or alternatively, we can use the local IP address.
	// (But since we can run multiple programs on the same PC, we also append the
	//  process ID)
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	//__________________________________________________________________
	//---------------------- PEER DISCOVERY ----------------------------
	//__________________________________________________________________

	// We make a channel for receiving updates on the id's of the peers that are
	//  alive on the network
	peerUpdateCh := make(chan peers.PeerUpdate)

	// We can disable/enable the transmitter after it has been started.
	// This could be used to signal that we are somehow "unavailable".
	peerTxEnable := make(chan bool)
	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)
 
	//__________________________________________________________________
	//------------- STARTER KOMMUNIKASJON MED HEARTBEATS ---------------
	//__________________________________________________________________

	// We make channels for sending and receiving our custom data types
	heartbeatTx := make(chan Heartbeat)
	heartbeatRx := make(chan Heartbeat)
	// ... and start the transmitter/receiver pair on some port
	// These functi
	//  start multiple transmitters/receivers on the same port.
	go bcast.Transmitter(16569, heartbeatTx)
	go bcast.Receiver(16569, heartbeatRx)

	//__________________________________________________________________
	//----------- SENDER DENNE NODEN SINE HEARTBEATS PERIODISK ---------
	//__________________________________________________________________

	// The example message. We just send one of these every second.
	go func() {
		HeartbeatMsg := Heartbeat{
			SenderID: id,
			ElevatorState: ElevatorState{
				Floor:    0,
				Dir:      true,
				DoorOpen: false,
				Idle:     true,
				Error:    false,
			},
			HallOrders: []Order{
				{Floor: 1, Direction: true, Status: OrderUnassigned, SyncState: SyncUnconfirmed, OwnerID: 1},
				{Floor: 4, Direction: false, Status: OrderAssigned, SyncState: SyncConfirmed, OwnerID: 2},
			},
			CabOrders: []Order{
				{Floor: 1, Direction: true, Status: OrderServed, SyncState: SyncConfirmed, OwnerID: 1},
				{Floor: 3, Direction: false, Status: OrderAssigned, SyncState: SyncUnconfirmed, OwnerID: 2},
			},
		}
		for {
			heartbeatTx <- HeartbeatMsg
			time.Sleep(1 * time.Second) //Endre til ønsket frekvens
		}
	}()

	//__________________________________________________________________
	//----------------  PRINTER INFORMASJON ----------------------------
	//__________________________________________________________________

	fmt.Println("Started")
	for {
		select {
		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

		case a := <-heartbeatRx:
			fmt.Printf("Received: %#v\n", a.SenderID)
		}
	}
}
 