package main

import (
	"log"
	"os"

	"github.com/TFMV/stride/cmd"
)

func main() {
	// Configure logger for detailed output.
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Set up a deferred function to recover from panics.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
			os.Exit(1)
		}
	}()

	if err := cmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
