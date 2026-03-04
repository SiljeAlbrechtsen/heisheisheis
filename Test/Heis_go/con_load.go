package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ConLoad(filename string, handler func(key, val string)) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Unable to open config file %s\n", filename)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "--") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimPrefix(parts[0], "--")
		val := parts[1]
		handler(strings.ToLower(key), strings.ToLower(val))
	}
}

func LoadConfig(filename string, cfg *Elevator) {
	ConLoad(filename, func(key, val string) {
		switch key {
		case "dooropenduration_s":
			fmt.Sscanf(val, "%f", &cfg.config.doorOpenDuration_s)
		}
	})
}
