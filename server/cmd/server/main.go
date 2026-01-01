package main

import (
	"log"
	"os"

	"squash-ladder/server"
)

func main() {
	// Get configuration from environment or use defaults
	dataPath := os.Getenv("LADDER_DATA_FILE")
	if dataPath == "" {
		dataPath = "data/transaction_log.jsonl"
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
		DataPath: dataPath,
		HTTPPort: httpPort,
		GRPCPort: grpcPort,
	}

	if err := server.Run(cfg); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
