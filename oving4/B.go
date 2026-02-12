package main

//package main

import (
	"fmt"
	"net"
	"os/exec"
	"time"
)

func main() {
	cmd := exec.Command( //Skriver til terminal
		"gnome-terminal", "--",
		"bash", "-lc",
		"go run A.go; echo; echo 'A ferdig (trykk Enter for å lukke)'; read",
	)

	cmd.Start() // trykker "enter". Kjører parallellet

	fmt.Println("B: A startet. Nå sender B data i loop:")

	serverIP := net.ParseIP("127.0.0.1") // Denne IP-adressen er local host
	serverPort := 20022

	// Kombinerer IP + Port til en UDP adresse
	serverAddr := &net.UDPAddr{
		IP:   serverIP,
		Port: serverPort,
	}

	// Oppretter UDP socket og kobler til server adresse
	conn, _ := net.DialUDP("udp", nil, serverAddr)

	defer conn.Close()

	// Lager melding til serveren
	message := []byte("hei fra ingrid og silje")
	for {

		fmt.Println("B: heartbeat")
		conn.Write(message)
		time.Sleep(1 * time.Second)
	}

	// Sender UDP pakke til server adresse

	//fmt.Println("Sent bytes", message)
}
