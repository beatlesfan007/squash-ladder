package handlers

import (
	"context"
	"sort"

	playerspb "squash-ladder/server/gen/players"
)

// PlayersHandler implements the PlayersService gRPC service
type PlayersHandler struct {
	playerspb.UnimplementedPlayersServiceServer
}

// NewPlayersHandler creates a new players handler
func NewPlayersHandler() *PlayersHandler {
	return &PlayersHandler{}
}

// ListPlayers returns all players ordered by rank
func (h *PlayersHandler) ListPlayers(ctx context.Context, req *playerspb.ListPlayersRequest) (*playerspb.ListPlayersResponse, error) {
	// Mock player data
	mockPlayers := []*playerspb.Player{
		{Id: 1, Name: "Alice Johnson", Rank: 1},
		{Id: 2, Name: "Bob Smith", Rank: 2},
		{Id: 3, Name: "Charlie Brown", Rank: 3},
		{Id: 4, Name: "Diana Prince", Rank: 4},
		{Id: 5, Name: "Eve Williams", Rank: 5},
		{Id: 6, Name: "Frank Miller", Rank: 6},
		{Id: 7, Name: "Grace Lee", Rank: 7},
		{Id: 8, Name: "Henry Davis", Rank: 8},
	}

	// Sort by rank to ensure correct order
	sort.Slice(mockPlayers, func(i, j int) bool {
		return mockPlayers[i].Rank < mockPlayers[j].Rank
	})

	return &playerspb.ListPlayersResponse{
		Players: mockPlayers,
	}, nil
}

