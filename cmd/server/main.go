package main

import (
	"log"

	"rosadisk-agent/internal/server"
)

func main() {
	srv := server.NewServer()

	log.Println("Starting server on :8080")
	if err := srv.Start(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
