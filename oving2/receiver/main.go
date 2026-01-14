package main

import (
	"fmt"
	"net"
)

func main() {
	addr := net.UDPAddr{
		IP:  net.ParseIP("0.0.0.0"),   //Hører på alle nettverksgrensesnitt på port 30000
		Port: 20022, // Porten på pcen vår. 
	}

	recvSock, _ := net.ListenUDP("udp", &addr) // Den oppretter UDP socket og binder til adressen over. 
		
	defer recvSock.Close()   //Lukker socket når main er ferdig

	buffer := make([]byte,1024)   //Her kommer det data fra serveren

	//localIP := getLocalIP()

	for {
		n, fromWho, _ := recvSock.ReadFromUDP(buffer) //Fyller buffer med date, og returnerer antall bytes n, og ip:port til fromWHo


		msg := string(buffer[:n])  // Gjøre buffer om til string 


		fmt.Printf("Mottatt fra %s: %s\n", fromWho.String(),msg,)  // skrivver ut melding fra server
	}

}




// IP addressen til server: 10.100.23.11:47102: 
// 47102 dette er port fra server