package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	fmt.Println("Program startet")

	addr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"), //Hører på alle nettverksgrensesnitt på port 30000
		Port: 20023,                    // Porten på pcen vår.
	}

	recvSock, _ := net.ListenUDP("udp", &addr) // Den oppretter UDP socket og binder til adressen over.

	defer recvSock.Close() //Lukker socket når main er ferdig

	buffer := make([]byte, 1024) //Her kommer det data fra serveren

	const ( //konstanter
		readTimeout   = 1500 * time.Millisecond // hvor lenge vi venter på én heartbeat
		missThreshold = 3                       // hvor mange på rad vi tåler
	)
	misses := 0

	for {
		// Deadline: hvis vi ikke får pakke innen readTimeout => timeout error
		_ = recvSock.SetReadDeadline(time.Now().Add(readTimeout))

		n, fromWho, err := recvSock.ReadFromUDP(buffer)
		if err != nil {
			// Sjekk om det er timeout
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				misses++
				fmt.Printf("A: timeout (%d/%d)\n", misses, missThreshold)

				if misses >= missThreshold {
					fmt.Println("A: B antas død -> takeover (eller avslutt)")
					return
				}
				continue
			}

		}

		// Fikk heartbeat -> reset misses
		misses = 0

		msg := string(buffer[:n])
		fmt.Printf("A: mottatt fra %s: %s\n", fromWho.String(), msg)
	}
}
