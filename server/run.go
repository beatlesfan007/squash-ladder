package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	ladderpb "squash-ladder/server/gen/ladder"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

// Config holds the configuration for the server
type Config struct {
	DataPath string
	HTTPPort string
	GRPCPort string
}

// Run starts the server with the given configuration.
// It blocks until the server fails or is stopped.
func Run(cfg Config) error {
	// Ensure data directory exists
	dataDir := filepath.Dir(cfg.DataPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}
	ladderModel, err := NewModel(cfg.DataPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ladder: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Create and register ladder service
	ladderService := NewLadderService(ladderModel)
	ladderpb.RegisterLadderServiceServer(grpcServer, ladderService)

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
			resp, err := ladderService.ListPlayers(r.Context(), &ladderpb.ListPlayersRequest{})
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

	// Start standard gRPC server (for specialized clients/testing)
	if cfg.GRPCPort != "" {
		go func() {
			lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
			if err != nil {
				log.Printf("failed to listen on :%s: %v", cfg.GRPCPort, err)
				return
			}
			log.Printf("Starting standard gRPC server on port %s", cfg.GRPCPort)
			if err := grpcServer.Serve(lis); err != nil {
				log.Printf("failed to serve gRPC on :%s: %v", cfg.GRPCPort, err)
			}
		}()
	}

	// Start HTTP server (serves both gRPC-Web and regular HTTP)
	log.Printf("Starting gRPC-Web server on port %s", cfg.HTTPPort)
	return http.ListenAndServe(fmt.Sprintf(":%s", cfg.HTTPPort), handler)
}
