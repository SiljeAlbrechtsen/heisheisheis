package main
// definere meldingstyper


// Sende-funksjon
func Transmitter(port int, chans ...interface{}) {
	// TODO:
	// sjekke om kanalene er gyldig (checkArgs)
	// gjøre om til JSON
	// broadcast på port
	// Kanskje: sende periodisk
}

// Mottak-funksjon

func Receiver(port int, chans ...interface{}) {
	// TODO:
	// sjekke om kanalene er gyldige (checkArgs)
	// høre på port
	// dekode JSON
	// sende på chans
}


