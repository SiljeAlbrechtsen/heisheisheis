// Send a message to the server:  "Connect to: " <your IP> ":" <your port> "\0"

// do not need IP, because we will set it to listening state
addr = new InternetAddress(localPort)
acceptSock = new Socket(tcp)

// You may not be able to use the same port twice when you restart the program, unless you set this option
acceptSock.setOption(REUSEADDR, true)
acceptSock.bind(addr)

loop {
    // backlog = Max number of pending connections waiting to connect()
    newSock = acceptSock.listen(backlog)

    // Spawn new thread to handle recv()/send() on newSock
}