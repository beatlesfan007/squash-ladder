package server

import (
	"os"
	"testing"

	ladderpb "squash-ladder/server/gen/ladder"
)

func createTempModel(t *testing.T) (*Model, string) {
	tmpFile, err := os.CreateTemp("", "ladder_log_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	m, err := NewModel(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create model: %v", err)
	}

	return m, tmpFile.Name()
}

func TestModel_AddPlayer(t *testing.T) {
	m, path := createTempModel(t)
	defer os.Remove(path)

	p, err := m.AddPlayer("Alice", "alice-id")
	if err != nil {
		t.Errorf("AddPlayer failed: %v", err)
	}

	if p.Name != "Alice" || p.Id != "alice-id" || p.Rank != 1 {
		t.Errorf("unexpected player data: %+v", p)
	}

	players := m.ListPlayers()
	if len(players) != 1 {
		t.Errorf("expected 1 player, got %d", len(players))
	}
}

func TestModel_RemovePlayer(t *testing.T) {
	m, path := createTempModel(t)
	defer os.Remove(path)

	m.AddPlayer("Alice", "alice-id")
	m.AddPlayer("Bob", "bob-id")
	m.AddPlayer("Charlie", "charlie-id")

	err := m.RemovePlayer("bob-id")
	if err != nil {
		t.Errorf("RemovePlayer failed: %v", err)
	}

	players := m.ListPlayers()
	if len(players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(players))
	}

	if players[0].Id != "alice-id" || players[0].Rank != 1 {
		t.Errorf("Alice rank incorrect: %+v", players[0])
	}
	if players[1].Id != "charlie-id" || players[1].Rank != 2 {
		t.Errorf("Charlie rank incorrect: %+v", players[1])
	}
}

func TestModel_ApplyMatchResult(t *testing.T) {
	m, path := createTempModel(t)
	defer os.Remove(path)

	m.AddPlayer("Alice", "alice")     // Rank 1
	m.AddPlayer("Bob", "bob")         // Rank 2
	m.AddPlayer("Charlie", "charlie") // Rank 3

	// Charlie (Rank 3) beats Alice (Rank 1)
	_, err := m.AddMatchResult("charlie", "alice", "charlie", []*ladderpb.SetScore{
		{Player1Points: 11, Player2Points: 5},
		{Player1Points: 11, Player2Points: 5},
		{Player1Points: 11, Player2Points: 5},
	})
	if err != nil {
		t.Fatalf("AddMatchResult failed: %v", err)
	}

	players := m.ListPlayers()
	// New Order: Charlie (1), Alice (2), Bob (3)
	if players[0].Id != "charlie" || players[0].Rank != 1 {
		t.Errorf("Charlie should be Rank 1, got %v", players[0])
	}
	if players[1].Id != "alice" || players[1].Rank != 2 {
		t.Errorf("Alice should be Rank 2, got %v", players[1])
	}
	if players[2].Id != "bob" || players[2].Rank != 3 {
		t.Errorf("Bob should be Rank 3, got %v", players[2])
	}
}

func TestModel_InvalidateMatchResult(t *testing.T) {
	m, path := createTempModel(t)
	defer os.Remove(path)

	m.AddPlayer("Alice", "alice")
	m.AddPlayer("Bob", "bob")

	txID, _ := m.AddMatchResult("bob", "alice", "bob", []*ladderpb.SetScore{
		{Player1Points: 11, Player2Points: 5},
		{Player1Points: 11, Player2Points: 5},
		{Player1Points: 11, Player2Points: 5},
	})

	// Bob should be #1 now
	if m.ListPlayers()[0].Id != "bob" {
		t.Fatal("Bob should be #1")
	}

	err := m.InvalidateMatchResult(txID)
	if err != nil {
		t.Fatalf("InvalidateMatchResult failed: %v", err)
	}

	// Alice should be #1 again
	if m.ListPlayers()[0].Id != "alice" {
		t.Fatal("Alice should be #1 after invalidation")
	}
}

func TestModel_GetRecentMatches(t *testing.T) {
	m, path := createTempModel(t)
	defer os.Remove(path)

	m.AddPlayer("Alice", "alice")
	m.AddPlayer("Bob", "bob")

	m.AddMatchResult("alice", "bob", "alice", []*ladderpb.SetScore{{Player1Points: 11, Player2Points: 0}, {Player1Points: 11, Player2Points: 0}, {Player1Points: 11, Player2Points: 0}})
	m.AddMatchResult("bob", "alice", "bob", []*ladderpb.SetScore{{Player2Points: 11, Player1Points: 0}, {Player2Points: 11, Player1Points: 0}, {Player2Points: 11, Player1Points: 0}})

	matches, err := m.GetRecentMatches(10)
	if err != nil {
		t.Fatalf("GetRecentMatches failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}

	// Should be in reverse order
	if matches[0].WinnerId != "bob" {
		t.Errorf("first match should be the latest one (winner: bob), got %s", matches[0].WinnerId)
	}
}

func TestModel_Persistence(t *testing.T) {
	m, path := createTempModel(t)
	defer os.Remove(path)

	m.AddPlayer("Alice", "alice")
	m.AddPlayer("Bob", "bob")
	m.AddMatchResult("bob", "alice", "bob", []*ladderpb.SetScore{
		{Player1Points: 11, Player2Points: 5},
		{Player1Points: 11, Player2Points: 5},
		{Player1Points: 11, Player2Points: 5},
	})

	// Load new model from same file
	m2, err := NewModel(path)
	if err != nil {
		t.Fatalf("failed to load model: %v", err)
	}

	players := m2.ListPlayers()
	if len(players) != 2 || players[0].Id != "bob" {
		t.Errorf("State not recovered correctly: %+v", players)
	}
}
