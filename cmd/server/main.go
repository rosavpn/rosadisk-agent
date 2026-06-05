package main

import (
	"log"

	"go.uber.org/zap"
	"rosadisk-agent/internal/server"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	srv := server.NewServer(logger)

	logger.Info("starting server", zap.String("addr", ":8080"))
	if err := srv.Start(":8080"); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}
