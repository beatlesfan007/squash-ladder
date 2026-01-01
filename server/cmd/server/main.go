package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	playerspb "squash-ladder/server/gen/players"
	"squash-ladder/server/handlers"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

func main() {
	// Initialize Ladder
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	ladder, err := handlers.NewLadder("data/transaction_log.jsonl")
	if err != nil {
		log.Fatalf("Failed to initialize ladder: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Create and register players service
	playersHandler := handlers.NewPlayersHandler(ladder)
	playerspb.RegisterPlayersServiceServer(grpcServer, playersHandler)

	// Wrap gRPC server with gRPC-Web
	wrappedGrpc := grpcweb.WrapServer(grpcServer)

	// Create HTTP handler with CORS support
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Grpc-Web, X-User-Agent, X-Grpc-Web-Type, Grpc-Timeout")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Serve JSON REST endpoint as fallback (for client compatibility)
		if r.URL.Path == "/api/players" && r.Method == "GET" {
			resp, err := playersHandler.ListPlayers(r.Context(), &playerspb.ListPlayersRequest{})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			// Convert proto response to JSON
			players := make([]map[string]interface{}, len(resp.Players))
			for i, p := range resp.Players {
				players[i] = map[string]interface{}{
					"id":   p.Id,
					"name": p.Name,
					"rank": p.Rank,
				}
			}
			jsonData := map[string]interface{}{
				"players": players,
			}
			json.NewEncoder(w).Encode(jsonData)
			return
		}

		// Serve gRPC-Web requests
		if wrappedGrpc.IsGrpcWebRequest(r) || wrappedGrpc.IsAcceptableGrpcCorsRequest(r) {
			wrappedGrpc.ServeHTTP(w, r)
			return
		}

		// For non-gRPC requests, return 404
		http.NotFound(w, r)
	})

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start standard gRPC server on port 9090 (for specialized clients/testing)
	go func() {
		lis, err := net.Listen("tcp", ":9090")
		if err != nil {
			log.Fatalf("failed to listen on :9090: %v", err)
		}
		log.Printf("Starting standard gRPC server on port 9090")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC on :9090: %v", err)
		}
	}()

	// Start HTTP server (serves both gRPC-Web and regular HTTP)
	log.Printf("Starting gRPC-Web server on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handler))
}
