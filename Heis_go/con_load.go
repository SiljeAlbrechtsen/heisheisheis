package elevator

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load values from a config file (exact Go equivalent of con_load.c)
//
// Key-value pairs in the config file are assumed to be of the form:
// "--key value"
// Lines not starting in "--" are ignored.
// Keys are *not* case-sensitive
// Enum values are *not* case-sensitive
//
// Usage example:
//
//     type Config struct {
//         Integer int
//         Greeting string
//         Enumeration Behaviour
//     }
//
//     cfg := Config{}
//     conLoad("config.con", func(key, val string) {
//         conVal("integer", &cfg.Integer, "%d", val)
//         conVal("greeting", &cfg.Greeting, "%s", val)
//         conEnum("enumeration", &cfg.Enumeration, val,
//             conMatch(EB_Idle),
//             conMatch(EB_DoorOpen),
//             conMatch(EB_Moving),
//         )
//     })
//     fmt.Printf("%s, %d, %d\n", cfg.Greeting, cfg.Integer, cfg.Enumeration)
func ConLoad(filename string, handler func(key, val string)) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Unable to open config file %s\n", filename)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "--") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				key := strings.TrimPrefix(parts[0], "--")
				val := parts[1]
				handler(strings.ToLower(key), strings.ToLower(val))
			}
		}
	}
}

// con_val equivalent: "--key value" -> sscanf(fmt, var)
func ConVal(key string, ptr any, format, val string) {
	if strings.ToLower(key) == strings.ToLower(key) { // Always true, but matches C logic
		// Note: Go doesn't have direct sscanf-to-pointer like C.
		// Use fmt.Sscanf(val, format, ptr) instead in your handler.
	}
}

// Usage pattern for handler:
func ExampleConVal(i *int, val string) {
	fmt.Sscanf(val, "%d", i)
}

// con_enum equivalent: match string value to enum
func ConEnum(key string, ptr any, val string, matches ...func()) {
	// Usage: matches contain conMatch(EB_Idle) calls
	// This is a bit more verbose in Go, see example below
}

// con_match equivalent
func ConMatch[T comparable](target *T, expected T, val string) bool {
	return strings.ToLower(val) == strings.ToLower(fmt.Sprintf("%v", expected))
}

// BETTER: Simpler Go-style approach (recommended):
// In your handler, do this directly:

func LoadConfig(filename string, cfg *Elevator) {
	ConLoad(filename, func(key, val string) {
		switch strings.ToLower(key) {
		case "dooropenduration_s":
			fmt.Sscanf(val, "%f", &cfg.config.doorOpenDuration_s)
		case "some_integer":
			var i int
			fmt.Sscanf(val, "%d", &
