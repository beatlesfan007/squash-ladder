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

		if s.ChallengerDefault || s.DefenderDefault {
			if !isLastSet {
				return 0, fmt.Errorf("defaulting player must happen in the final set")
			}
			if s.ChallengerDefault && s.DefenderDefault {
				return 0, fmt.Errorf("both players cannot default")
			}
			// Valid default in final set
			if s.ChallengerDefault {
				// Challenger defaulted, Defender wins
				return 2, nil
			}
			// Defender defaulted, Challenger wins
			return 1, nil
		}

		challengerPoints := int(s.ChallengerPoints)
		defenderPoints := int(s.DefenderPoints)

		// Validate set rules: to 11, win by 2
		if challengerPoints < 0 || defenderPoints < 0 {
			return 0, fmt.Errorf("scores cannot be negative")
		}

		if challengerPoints < 11 && defenderPoints < 11 {
			return 0, fmt.Errorf("set must go to at least 11: %d-%d", challengerPoints, defenderPoints)
		}

		diff := challengerPoints - defenderPoints
		if diff < 0 {
			diff = -diff
		}

		if diff < 2 {
			return 0, fmt.Errorf("must win by 2 points: %d-%d", challengerPoints, defenderPoints)
		}

		// Determine winner of set
		if challengerPoints > defenderPoints {
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
	// 1 = Challenger, 2 = Defender
	if winnerIdx == 1 && req.WinnerId != req.ChallengerId {
		return &ladderpb.AddMatchResultResponse{Success: false}, fmt.Errorf("scores indicate challenger won, but winner_id does not match challenger")
	}
	if winnerIdx == 2 && req.WinnerId != req.DefenderId {
		return &ladderpb.AddMatchResultResponse{Success: false}, fmt.Errorf("scores indicate defender won, but winner_id does not match defender")
	}

	txID, err := h.model.AddMatchResult(req.ChallengerId, req.DefenderId, req.WinnerId, req.SetScores)
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
