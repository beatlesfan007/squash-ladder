package server

import (
	"context"
	"os"
	"testing"

	ladderpb "squash-ladder/server/gen/ladder"
)

func TestValidateScore(t *testing.T) {
	tests := []struct {
		name       string
		scores     []*ladderpb.SetScore
		wantWinner int
		wantErr    bool
	}{
		{
			name: "Valid 3-0 win",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: 11, DefenderPoints: 5},
				{ChallengerPoints: 11, DefenderPoints: 5},
				{ChallengerPoints: 11, DefenderPoints: 5},
			},
			wantWinner: 1,
			wantErr:    false,
		},
		{
			name: "Valid 3-2 win",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: 11, DefenderPoints: 5},
				{ChallengerPoints: 5, DefenderPoints: 11},
				{ChallengerPoints: 11, DefenderPoints: 5},
				{ChallengerPoints: 5, DefenderPoints: 11},
				{ChallengerPoints: 11, DefenderPoints: 5},
			},
			wantWinner: 1,
			wantErr:    false,
		},
		{
			name: "Must win by 2",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: 11, DefenderPoints: 10},
			},
			wantWinner: 0,
			wantErr:    true,
		},
		{
			name: "Set must go to 11",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: 5, DefenderPoints: 2},
			},
			wantWinner: 0,
			wantErr:    true,
		},
		{
			name: "Default in final set",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: 11, DefenderPoints: 5},
				{ChallengerDefault: true},
			},
			wantWinner: 2,
			wantErr:    false,
		},
		{
			name: "Default not in final set",
			scores: []*ladderpb.SetScore{
				{ChallengerDefault: true},
				{ChallengerPoints: 11, DefenderPoints: 5},
			},
			wantWinner: 0,
			wantErr:    true,
		},
		{
			name: "Negative scores",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: -1, DefenderPoints: 5},
			},
			wantWinner: 0,
			wantErr:    true,
		},
		{
			name: "Both players default",
			scores: []*ladderpb.SetScore{
				{ChallengerDefault: true, DefenderDefault: true},
			},
			wantWinner: 0,
			wantErr:    true,
		},
		{
			name: "Player 2 defaults",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: 11, DefenderPoints: 5},
				{DefenderDefault: true},
			},
			wantWinner: 1,
			wantErr:    false,
		},
		{
			name: "Too many sets",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: 11, DefenderPoints: 5},
				{ChallengerPoints: 11, DefenderPoints: 5},
				{ChallengerPoints: 11, DefenderPoints: 5},
				{ChallengerPoints: 11, DefenderPoints: 5}, // 4th set win
			},
			wantWinner: 0,
			wantErr:    true,
		},
		{
			name: "Not enough sets",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: 11, DefenderPoints: 5},
				{ChallengerPoints: 11, DefenderPoints: 5},
			},
			wantWinner: 0,
			wantErr:    true,
		},
		{
			name: "Extended play win",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: 12, DefenderPoints: 10},
				{ChallengerPoints: 15, DefenderPoints: 13},
				{ChallengerPoints: 11, DefenderPoints: 9},
			},
			wantWinner: 1,
			wantErr:    false,
		},
		{
			name: "Extended play invalid win",
			scores: []*ladderpb.SetScore{
				{ChallengerPoints: 12, DefenderPoints: 11},
			},
			wantWinner: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			winner, err := ValidateScore(tt.scores)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if winner != tt.wantWinner {
				t.Errorf("ValidateScore() winner = %v, want %v", winner, tt.wantWinner)
			}
		})
	}
}

func TestLadderService_AddMatchResult(t *testing.T) {
	m, path := createTempModel(t)
	defer os.Remove(path)

	m.AddPlayer("Alice", "alice")
	m.AddPlayer("Bob", "bob")

	svc := NewLadderService(m)

	// Valid result
	resp, err := svc.AddMatchResult(context.Background(), &ladderpb.AddMatchResultRequest{
		ChallengerId: "alice",
		DefenderId:   "bob",
		WinnerId:     "bob",
		SetScores: []*ladderpb.SetScore{
			{DefenderPoints: 11, ChallengerPoints: 0},
			{DefenderPoints: 11, ChallengerPoints: 0},
			{DefenderPoints: 11, ChallengerPoints: 0},
		},
	})

	if err != nil {
		t.Fatalf("AddMatchResult failed: %v", err)
	}
	if !resp.Success {
		t.Error("expected success = true")
	}

	// Inconsistent winner
	_, err = svc.AddMatchResult(context.Background(), &ladderpb.AddMatchResultRequest{
		ChallengerId: "alice",
		DefenderId:   "bob",
		WinnerId:     "alice", // Alice didn't win according to scores
		SetScores: []*ladderpb.SetScore{
			{DefenderPoints: 11, ChallengerPoints: 0},
			{DefenderPoints: 11, ChallengerPoints: 0},
			{DefenderPoints: 11, ChallengerPoints: 0},
		},
	})
	if err == nil {
		t.Error("expected error for inconsistent winner")
	}
}

func TestLadderService_ListRecentMatches(t *testing.T) {
	m, path := createTempModel(t)
	defer os.Remove(path)

	m.AddPlayer("Alice", "alice")
	m.AddPlayer("Bob", "bob")
	m.AddMatchResult("alice", "bob", "alice", []*ladderpb.SetScore{
		{ChallengerPoints: 11, DefenderPoints: 0},
		{ChallengerPoints: 11, DefenderPoints: 0},
		{ChallengerPoints: 11, DefenderPoints: 0},
	})

	svc := NewLadderService(m)
	resp, err := svc.ListRecentMatches(context.Background(), &ladderpb.ListRecentMatchesRequest{Limit: 10})
	if err != nil {
		t.Fatalf("ListRecentMatches failed: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Errorf("expected 1 match, got %d", len(resp.Results))
	}
}
