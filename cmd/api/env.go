package main

import (
	"bufio"
	"os"
	"strings"
)

// loadDotenv loads .env key-value paires into the environmen without overwriting existing variables.
func loadDotenv(path string) {
	// Skip if file cannot be opened
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())

		// Ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Spilit into key and value by first '='
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		// Clean key and strp surrounding quotes from value
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key == "" {
			continue
		}

		// Set variable if not already present in the environment
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}

	if err := sc.Err(); err != nil {
		os.Stderr.WriteString("loadDotenv: " + err.Error() + "\n")
	}
}
