package main

import (
	//"bufio"
	"fmt"
	"net"
)

func main() {

	serverIP := net.ParseIP("10.100.23.11")
	serverPort := 33546 // Separerer meldinger med \0 istedenfor at de har fast lengde.

	// Kombinerer IP + Port til en TCP adresse
	serverAddr := &net.TCPAddr{
		IP:   serverIP,
		Port: serverPort,
	}

	// Kobler til server
	conn, _ := net.DialTCP("tcp", nil, serverAddr)

	defer conn.Close() // Lukker tilkoblingen automatisk når main er ferdig

	buffer := make([]byte, 1024) // TCP kan bare lese data inn i et buffer

	n, _ := conn.Read(buffer) // Returnerer antall bytes som ble lest fra buffer

	fmt.Println(string(buffer[:n])) // Printer det den leser

	message := []byte("hei\000")

	conn.Write(message) // Sender alle bytes i message over TCP

	buf := make([]byte, 1024) // Lager en ny buffer som TCP bruker til å motta data. Echo buffer
	n, _ = conn.Read(buf)
	fmt.Println(string(buf[:n]))

	// Definerer hvor vi skal lytte. Client -> Server
	listenaddr := net.TCPAddr{
		IP:   net.ParseIP("10.100.23.33"),
		Port: 20023,
	}

	//Start å lytte (dette gjør oss til SERVER)
	listener, _ := net.ListenTCP("tcp", &listenaddr)

	defer listener.Close()

	message = []byte("Connect to: 10.100.23.33:20023\000") // Meldingen vi vil sende
	conn.Write(message)                                    // Sender meldingen

	con, _ := listener.AcceptTCP() // Venter (blokkerer) til serveren kobler seg til

	defer con.Close()

	fmt.Println("Noen koblet seg til!")

	buf = make([]byte, 1024)
	n, _ = con.Read(buf)

	fmt.Println("Mottatt:", string(buf[:n]))

	message = []byte("Hei Ingrid\000") // Meldingen vi vil sende
	con.Write(message)                 // sender meldingen

	// Leser echo fra serveren
	buf = make([]byte, 1024)
	n, _ = con.Read(buf)

	fmt.Println("Mottatt:", string(buf[:n]))
}
