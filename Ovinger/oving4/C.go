package main

//package main

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"time"
)

func fra_B(num int) {
	cmd := exec.Command( //Skriver til terminal
		"gnome-terminal", "--",
		"bash", "-lc",
		"go run C.go",
	)

	cmd.Start() // trykker "enter". Kjører parallellet

	fmt.Println("B: A startet. Nå sender B data i loop:")

	serverIP := net.ParseIP("127.0.0.1") // Denne IP-adressen er local host
	serverPort := 20023

	// Kombinerer IP + Port til en UDP adresse
	serverAddr := &net.UDPAddr{
		IP:   serverIP,
		Port: serverPort,
	}

	// Oppretter UDP socket og kobler til server adresse
	conn, _ := net.DialUDP("udp", nil, serverAddr)

	defer conn.Close()

	time.Sleep(1 * time.Second)

	for i := 0; i < 5; i++ {
		msg := []byte(fmt.Sprintf("%d", num))
		fmt.Println(num)
		conn.Write(msg)
		num++
		time.Sleep(1 * time.Second)
	}

	// Sender UDP pakke til server adresse

	//fmt.Println("Sent bytes", message)
}

func main() {
	num := 0
	fmt.Println("Program startet")

	addr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"), //Hører på alle nettverksgrensesnitt på port 30000
		Port: 20023,                    // Porten på pcen vår.
	}

	recvSock, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err) // eller fmt.Println("ListenUDP feil:", err); return
	}

	defer recvSock.Close() //Lukker socket når main er ferdig

	buffer := make([]byte, 1024) //Her kommer det data fra serveren

	const ( //konstanter
		readTimeout   = 1500 * time.Millisecond // hvor lenge vi venter på én heartbeat
		missThreshold = 1                       // hvor mange på rad vi tåler
	)
	misses := 0

	for {
		// Deadline: hvis vi ikke får pakke innen readTimeout => timeout error
		_ = recvSock.SetReadDeadline(time.Now().Add(readTimeout))

		//n, fromWho, err := recvSock.ReadFromUDP(buffer)
		n, _, err := recvSock.ReadFromUDP(buffer)

		if err != nil {
			// Sjekk om det er timeout
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				misses++
				fmt.Printf("A: timeout (%d/%d)\n", misses, missThreshold)

				if misses >= missThreshold {
					fmt.Println("A: B antas død -> takeover")
					recvSock.Close() // <-- VIKTIG: frigjør porten
					num++
					fra_B(num) // nå kan en ny backup starte og binde porten
					return
				}
				continue
			}

		}

		// Fikk heartbeat -> reset misses
		misses = 0

		msg := string(buffer[:n])
		//fmt.Printf("Copy  %s: %s\n", fromWho.String(), msg)
		num, err = strconv.Atoi(msg)
		if err != nil {
			fmt.Println("Kunne ikke konvertere:", err)
			return
		}
	}
}
