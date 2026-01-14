package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	n := 22 // <-- workspace-nummeret deres
	serverIP := "10.100.23.11"
	serverPort := 20000 + n

	serverAddr := &net.UDPAddr{
		IP:   net.ParseIP(serverIP),
		Port: serverPort,
	}

	// Lokal port kan være 0 (da velger OS en ledig port),
	// men siden vi leser på SAMME conn, får vi uansett svaret.
	localAddr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Skriv hvilken lokal adresse/port vi faktisk fikk
	fmt.Println("Lokal socket:", conn.LocalAddr().String())
	fmt.Println("Sender til:", serverAddr.String())

	msg := []byte("hei fra ingrid og silje")
	if _, err := conn.WriteToUDP(msg, serverAddr); err != nil {
		panic(err)
	}
	fmt.Println("Sendt:", string(msg))

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	buf := make([]byte, 2048)
	nRead, from, err := conn.ReadFromUDP(buf)
	if err != nil {
		fmt.Println("Fikk ikke svar (timeout/feil):", err)
		return
	}

	fmt.Printf("Mottatt fra %s: %s\n", from.String(), string(buf[:nRead]))
}
