package main

import (
	"log"
	"os"

	"squash-ladder/server"
)

func main() {
	// Get configuration from environment or use defaults
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Default to local postgres via docker-compose
		dbURL = "postgres://postgres:password@localhost:5432/squash_ladder?sslmode=disable"
	}

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}

	cfg := server.Config{
		DatabaseURL: dbURL,
		HTTPPort:    httpPort,
		GRPCPort:    grpcPort,
	}

	if err := server.Run(cfg); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
