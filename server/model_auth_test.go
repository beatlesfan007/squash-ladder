package server

import (
	"database/sql"
	"testing"
)

func TestModel_InvitationFlow(t *testing.T) {
	m := setupTestDB(t)

	// 1. Admin adds a player (mocked admin, effectively just calling AddPlayer directly as the service would)
	player, err := m.AddPlayer("New Guy", "new-guy-id")
	if err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// 2. Generate Invite
	token, err := m.CreateInvitation(player.Id)
	if err != nil {
		t.Fatalf("CreateInvitation failed: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateInvite returned empty token")
	}

	// 3. Verify Invite exists in DB
	var dbPlayerID string
	err = m.Db.QueryRow("SELECT player_id FROM invitations WHERE token = $1", token).Scan(&dbPlayerID)
	if err != nil {
		t.Fatalf("Invitation not found in DB: %v", err)
	}
	if dbPlayerID != player.Id {
		t.Errorf("Invitation linked to wrong player: got %s, want %s", dbPlayerID, player.Id)
	}

	// 4. Claim Player with User ID "user-123"
	claimedID, err := m.ClaimPlayer("user-123", token)
	if err != nil {
		t.Fatalf("ClaimPlayer failed: %v", err)
	}
	if claimedID != player.Id {
		t.Errorf("ClaimPlayer returned wrong ID: got %s, want %s", claimedID, player.Id)
	}

	// 5. Verify Player has user_id set
	var userID sql.NullString
	err = m.Db.QueryRow("SELECT user_id FROM players WHERE id = $1", player.Id).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to fetch player user_id: %v", err)
	}
	if !userID.Valid || userID.String != "user-123" {
		t.Errorf("Player user_id not updated correctly: got %v, want user-123", userID)
	}

	// 6. Verify Invitation is deleted
	err = m.Db.QueryRow("SELECT player_id FROM invitations WHERE token = $1", token).Scan(&dbPlayerID)
	if err != sql.ErrNoRows {
		t.Errorf("Invitation should be deleted after claim, but found it (err: %v)", err)
	}

	// 7. Try to claim again (Server Logic Check)
	// Although the token is deleted, if we reused it somehow or generated another one for the same player?
	// Let's try to generate another invite for the now-claimed player.
	// The system likely allows generating multiple invites?
	// But `ClaimPlayer` checks if `user_id` is already set.

	token2, err := m.CreateInvitation(player.Id)
	if err != nil {
		t.Fatalf("CreateInvitation 2 failed: %v", err)
	}

	_, err = m.ClaimPlayer("other-user", token2)
	if err == nil {
		t.Error("ClaimPlayer should fail if player is already claimed")
	} else if err.Error() != "player already claimed" {
		t.Errorf("Unexpected error for claimed player: %v", err)
	}
}
