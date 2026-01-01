package handlers

import (
	"context"

	playerspb "squash-ladder/server/gen/players"
)

// PlayersHandler implements the PlayersService gRPC service
type PlayersHandler struct {
	playerspb.UnimplementedPlayersServiceServer
	ladder *Ladder
}

// NewPlayersHandler creates a new players handler
func NewPlayersHandler(ladder *Ladder) *PlayersHandler {
	return &PlayersHandler{
		ladder: ladder,
	}
}

// ListPlayers returns all players ordered by rank
func (h *PlayersHandler) ListPlayers(ctx context.Context, req *playerspb.ListPlayersRequest) (*playerspb.ListPlayersResponse, error) {
	players := h.ladder.ListPlayers()
	return &playerspb.ListPlayersResponse{
		Players: players,
	}, nil
}

// AddPlayer adds a new player
func (h *PlayersHandler) AddPlayer(ctx context.Context, req *playerspb.AddPlayerRequest) (*playerspb.AddPlayerResponse, error) {
	player, err := h.ladder.AddPlayer(req.Name, req.PlayerId)
	if err != nil {
		return nil, err
	}
	return &playerspb.AddPlayerResponse{Player: player}, nil
}

// RemovePlayer removes a player
func (h *PlayersHandler) RemovePlayer(ctx context.Context, req *playerspb.RemovePlayerRequest) (*playerspb.RemovePlayerResponse, error) {
	err := h.ladder.RemovePlayer(req.PlayerId)
	if err != nil {
		return &playerspb.RemovePlayerResponse{Success: false}, err
	}
	return &playerspb.RemovePlayerResponse{Success: true}, nil
}

// AddMatchResult records a match result
func (h *PlayersHandler) AddMatchResult(ctx context.Context, req *playerspb.AddMatchResultRequest) (*playerspb.AddMatchResultResponse, error) {
	txID, err := h.ladder.AddMatchResult(req.Player1Id, req.Player2Id, req.WinnerId, req.SetScores)
	if err != nil {
		return &playerspb.AddMatchResultResponse{Success: false}, err
	}
	return &playerspb.AddMatchResultResponse{Success: true, TransactionId: txID}, nil
}

// InvalidateMatchResult invalidates a match result
func (h *PlayersHandler) InvalidateMatchResult(ctx context.Context, req *playerspb.InvalidateMatchResultRequest) (*playerspb.InvalidateMatchResultResponse, error) {
	err := h.ladder.InvalidateMatchResult(req.TransactionId)
	if err != nil {
		return &playerspb.InvalidateMatchResultResponse{Success: false}, err
	}
	return &playerspb.InvalidateMatchResultResponse{Success: true}, nil
}
