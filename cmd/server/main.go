package main

import (
	"log"

	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/server"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	db, err := database.InitDB("/var/lib/rosadisk-agent/state.db")
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", zap.Error(err))
		}
	}()

	srv := server.NewServer(logger, db)

	logger.Info("starting server", zap.String("addr", ":8080"))
	if err := srv.Start(":8080"); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}
