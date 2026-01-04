package server

import (
	"testing"
)

func TestInvitationFlow(t *testing.T) {
	dbURL := "postgres://postgres:password@localhost:5432/squash_ladder_test?sslmode=disable"
	m, err := NewModel(dbURL)
	if err != nil {
		t.Skip("Skipping DB test: ", err)
		return
	}

	// Clean up
	m.Db.Exec("TRUNCATE players, invitations CASCADE")

	// 1. Add a player
	p, err := m.AddPlayer("Test Player", "test-p1")
	if err != nil {
		t.Fatalf("failed to add player: %v", err)
	}

	// 2. Generate Invite
	token, err := m.CreateInvitation(p.Id)
	if err != nil {
		t.Fatalf("failed to create invitation: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}

	// 3. Claim Player
	userID := "supabase-user-123"
	claimedID, err := m.ClaimPlayer(userID, token)
	if err != nil {
		t.Fatalf("failed to claim player: %v", err)
	}

	if claimedID != p.Id {
		t.Errorf("expected player ID %s, got %s", p.Id, claimedID)
	}

	// 4. Verify in DB
	var dbUserID string
	err = m.Db.QueryRow("SELECT user_id FROM players WHERE id = $1", p.Id).Scan(&dbUserID)
	if err != nil {
		t.Fatalf("failed to query player: %v", err)
	}
	if dbUserID != userID {
		t.Errorf("expected user_id %s, got %s", userID, dbUserID)
	}

	// 5. Verify invitation is gone
	var count int
	m.Db.QueryRow("SELECT COUNT(*) FROM invitations WHERE token = $1", token).Scan(&count)
	if count != 0 {
		t.Errorf("expected invitation to be deleted, found %d", count)
	}
}
