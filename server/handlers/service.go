package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	ladderpb "squash-ladder/server/gen/ladder"
	"squash-ladder/server/model"
)

// LadderService implements the LadderService gRPC service
type LadderService struct {
	ladderpb.UnimplementedLadderServiceServer
	model *model.Model
}

// NewLadderService creates a new ladder service handler
func NewLadderService(m *model.Model) *LadderService {
	return &LadderService{
		model: m,
	}
}

// ListPlayers returns all players ordered by rank
func (h *LadderService) ListPlayers(ctx context.Context, req *ladderpb.ListPlayersRequest) (*ladderpb.ListPlayersResponse, error) {
	players := h.model.ListPlayers()
	return &ladderpb.ListPlayersResponse{
		Players: players,
	}, nil
}

// AddPlayer adds a new player
func (h *LadderService) AddPlayer(ctx context.Context, req *ladderpb.AddPlayerRequest) (*ladderpb.AddPlayerResponse, error) {
	player, err := h.model.AddPlayer(req.Name, req.PlayerId)
	if err != nil {
		return nil, err
	}
	return &ladderpb.AddPlayerResponse{Player: player}, nil
}

// RemovePlayer removes a player
func (h *LadderService) RemovePlayer(ctx context.Context, req *ladderpb.RemovePlayerRequest) (*ladderpb.RemovePlayerResponse, error) {
	err := h.model.RemovePlayer(req.PlayerId)
	if err != nil {
		return &ladderpb.RemovePlayerResponse{Success: false}, err
	}
	return &ladderpb.RemovePlayerResponse{Success: true}, nil
}

// ValidateScore validates squash scoring rules
func ValidateScore(setScores []string) (bool, error) {
	p1Sets := 0
	p2Sets := 0

	for _, s := range setScores {
		parts := strings.Split(s, "-")
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid score format: %s", s)
		}

		// Check for default
		if parts[0] == "D" || parts[1] == "D" {
			// Logic for default: one player defaults, other wins set?
			// The requirements say: 7(D) is valid. If defaulted, it's the last set.
			return true, nil
		}

		p1Points, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		p2Points, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))

		if err1 != nil || err2 != nil {
			return false, fmt.Errorf("invalid points: %s", s)
		}

		// Validate set rules: to 11, win by 2
		if p1Points < 11 && p2Points < 11 {
			return false, fmt.Errorf("set must go to at least 11: %s", s)
		}

		diff := p1Points - p2Points
		if diff < 0 {
			diff = -diff
		}

		if diff < 2 {
			return false, fmt.Errorf("must win by 2 points: %s", s)
		}

		// Determine winner of set
		if p1Points > p2Points {
			p1Sets++
		} else {
			p2Sets++
		}
	}

	// Best of 5 (first to 3)
	if p1Sets > 3 || p2Sets > 3 {
		return false, fmt.Errorf("too many sets won")
	}

	return true, nil
}

// AddMatchResult records a match result
func (h *LadderService) AddMatchResult(ctx context.Context, req *ladderpb.AddMatchResultRequest) (*ladderpb.AddMatchResultResponse, error) {
	// Validate score
	if valid, err := ValidateScore(req.SetScores); !valid {
		return &ladderpb.AddMatchResultResponse{Success: false}, err
	}

	txID, err := h.model.AddMatchResult(req.Player1Id, req.Player2Id, req.WinnerId, req.SetScores)
	if err != nil {
		return &ladderpb.AddMatchResultResponse{Success: false}, err
	}
	return &ladderpb.AddMatchResultResponse{Success: true, TransactionId: txID}, nil
}

// InvalidateMatchResult invalidates a match result
func (h *LadderService) InvalidateMatchResult(ctx context.Context, req *ladderpb.InvalidateMatchResultRequest) (*ladderpb.InvalidateMatchResultResponse, error) {
	err := h.model.InvalidateMatchResult(req.TransactionId)
	if err != nil {
		return &ladderpb.InvalidateMatchResultResponse{Success: false}, err
	}
	return &ladderpb.InvalidateMatchResultResponse{Success: true}, nil
}
