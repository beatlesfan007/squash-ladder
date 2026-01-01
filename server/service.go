package server

import (
	"context"
	"fmt"

	ladderpb "squash-ladder/server/gen/ladder"
)

// LadderService implements the LadderService gRPC service
type LadderService struct {
	ladderpb.UnimplementedLadderServiceServer
	model *Model
}

// NewLadderService creates a new ladder service handler
func NewLadderService(m *Model) *LadderService {
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

// ValidateScore validates squash scoring rules and returns the winner (1 or 2)
func ValidateScore(setScores []*ladderpb.SetScore) (int, error) {
	p1Sets := 0
	p2Sets := 0

	for i, s := range setScores {
		isLastSet := i == len(setScores)-1

		if s.Player1Default || s.Player2Default {
			if !isLastSet {
				return 0, fmt.Errorf("defaulting player must happen in the final set")
			}
			if s.Player1Default && s.Player2Default {
				return 0, fmt.Errorf("both players cannot default")
			}
			// Valid default in final set
			if s.Player1Default {
				// Player 1 defaulted, Player 2 wins
				return 2, nil
			}
			// Player 2 defaulted, Player 1 wins
			return 1, nil
		}

		p1Points := int(s.Player1Points)
		p2Points := int(s.Player2Points)

		// Validate set rules: to 11, win by 2
		if p1Points < 0 || p2Points < 0 {
			return 0, fmt.Errorf("scores cannot be negative")
		}

		if p1Points < 11 && p2Points < 11 {
			return 0, fmt.Errorf("set must go to at least 11: %d-%d", p1Points, p2Points)
		}

		diff := p1Points - p2Points
		if diff < 0 {
			diff = -diff
		}

		if diff < 2 {
			return 0, fmt.Errorf("must win by 2 points: %d-%d", p1Points, p2Points)
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
		return 0, fmt.Errorf("too many sets won")
	}

	if p1Sets == 3 {
		return 1, nil
	}
	if p2Sets == 3 {
		return 2, nil
	}

	return 0, fmt.Errorf("match must have a clear winner (first to 3 sets)")
}

// AddMatchResult records a match result
func (h *LadderService) AddMatchResult(ctx context.Context, req *ladderpb.AddMatchResultRequest) (*ladderpb.AddMatchResultResponse, error) {
	// Validate score
	// Validate score covers defaults and calculates winner
	winnerIdx, err := ValidateScore(req.SetScores)
	if err != nil {
		return &ladderpb.AddMatchResultResponse{Success: false}, err
	}

	// Double check winner matches the score calculation
	if winnerIdx == 1 && req.WinnerId != req.Player1Id {
		return &ladderpb.AddMatchResultResponse{Success: false}, fmt.Errorf("scores indicate player 1 won, but winner_id does not match player 1")
	}
	if winnerIdx == 2 && req.WinnerId != req.Player2Id {
		return &ladderpb.AddMatchResultResponse{Success: false}, fmt.Errorf("scores indicate player 2 won, but winner_id does not match player 2")
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

// ListRecentMatches returns the last n matches
func (h *LadderService) ListRecentMatches(ctx context.Context, req *ladderpb.ListRecentMatchesRequest) (*ladderpb.ListRecentMatchesResponse, error) {
	matches, err := h.model.GetRecentMatches(req.Limit)
	if err != nil {
		return nil, err
	}
	return &ladderpb.ListRecentMatchesResponse{
		Results: matches,
	}, nil
}
