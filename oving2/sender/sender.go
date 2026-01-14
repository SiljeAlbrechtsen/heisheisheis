package main

import (
	//"fmt"
	"net"
)

func main() {
	
	
	serverIP :=  net.ParseIP("10.100.23.11")   
	serverPort := 20022

	// Kombinerer IP + Port til en UDP adresse
	serverAddr := &net.UDPAddr{
		IP: serverIP,
		Port: serverPort,
	}
	
	// Oppretter UDP socket og kobler til server adresse
	conn, _:= net.DialUDP("udp", nil, serverAddr)
	
	defer conn.Close()

	// Lager melding til serveren
	message := []byte("hei fra ingrid og silje")

	// Sender UDP pakke til server adresse
	conn.Write(message)


	//fmt.Println("Sent bytes", message)
}

