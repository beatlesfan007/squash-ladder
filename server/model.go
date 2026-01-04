package server

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	ladderpb "squash-ladder/server/gen/ladder"
	storagepb "squash-ladder/server/gen/storage"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
	"google.golang.org/protobuf/proto"
)

// Model manages the state of the squash ladder using PostgreSQL
type Model struct {
	mu sync.RWMutex
	Db *sql.DB
}

// NewModel creates a new model and connects to the database
func NewModel(dbURL string) (*Model, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to init schema: %v", err)
	}

	return &Model{
		Db: db,
	}, nil
}

func initSchema(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS players (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			rank INTEGER NOT NULL,
			user_id VARCHAR(255)
		);`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id VARCHAR(255) PRIMARY KEY,
			type INTEGER NOT NULL,
			timestamp_ms BIGINT NOT NULL,
			payload BYTEA NOT NULL,
			player_ranks BYTEA NOT NULL,
			is_invalidated BOOLEAN DEFAULT FALSE
		);`,
		`CREATE TABLE IF NOT EXISTS invitations (
			token VARCHAR(255) PRIMARY KEY,
			player_id VARCHAR(255) REFERENCES players(id) ON DELETE CASCADE,
			expires_at BIGINT NOT NULL
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// Helper to convert storage players to ladder players
func storageToLadder(sPlayers []*storagepb.PlayerStorage) []*ladderpb.Player {
	lPlayers := make([]*ladderpb.Player, len(sPlayers))
	for i, sp := range sPlayers {
		lPlayers[i] = &ladderpb.Player{
			Id:   sp.Id,
			Name: sp.Name,
			Rank: sp.Rank,
		}
	}
	return lPlayers
}

// Helper to convert ladder players to storage players
func ladderToStorage(lPlayers []*ladderpb.Player) []*storagepb.PlayerStorage {
	sPlayers := make([]*storagepb.PlayerStorage, len(lPlayers))
	for i, lp := range lPlayers {
		sPlayers[i] = &storagepb.PlayerStorage{
			Id:   lp.Id,
			Name: lp.Name,
			Rank: lp.Rank,
		}
	}
	return sPlayers
}

// CurrentState queries the players table for the current state
func (m *Model) CurrentState() ([]*ladderpb.Player, error) {
	rows, err := m.Db.Query("SELECT id, name, rank FROM players ORDER BY rank ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []*ladderpb.Player
	for rows.Next() {
		var p ladderpb.Player
		if err := rows.Scan(&p.Id, &p.Name, &p.Rank); err != nil {
			return nil, err
		}
		players = append(players, &p)
	}
	return players, nil
}

// applyTransactionLogic calculates the NEW player state based on a transaction type and payload.
// This is pure logic, separated from DB I/O.
func (m *Model) applyTransactionLogic(txType storagepb.TransactionType, payload interface{}, currentPlayers []*ladderpb.Player) ([]*ladderpb.Player, error) {
	// Deep copy players
	players := make([]*ladderpb.Player, len(currentPlayers))
	for i, p := range currentPlayers {
		players[i] = &ladderpb.Player{
			Id:   p.Id,
			Name: p.Name,
			Rank: p.Rank,
		}
	}

	switch txType {
	case storagepb.TransactionType_ADD_PLAYER:
		p, ok := payload.(*storagepb.AddPlayerStorage)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for ADD_PLAYER")
		}
		// Check duplicates
		for _, pl := range players {
			if pl.Id == p.PlayerId {
				return nil, fmt.Errorf("player ID already exists")
			}
		}
		newPlayer := &ladderpb.Player{
			Id:   p.PlayerId,
			Name: p.Name,
			Rank: int32(len(players) + 1),
		}
		players = append(players, newPlayer)

	case storagepb.TransactionType_REMOVE_PLAYER:
		p, ok := payload.(*storagepb.RemovePlayerStorage)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for REMOVE_PLAYER")
		}
		idx := -1
		for i, pl := range players {
			if pl.Id == p.PlayerId {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, fmt.Errorf("player not found")
		}
		players = append(players[:idx], players[idx+1:]...)
		// Re-rank
		for i := idx; i < len(players); i++ {
			players[i].Rank = int32(i + 1)
		}

	case storagepb.TransactionType_MATCH_RESULT:
		p, ok := payload.(*storagepb.MatchResultStorage)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for MATCH_RESULT")
		}

		challengerIdx := -1
		defenderIdx := -1
		for i, pl := range players {
			if pl.Id == p.ChallengerId {
				challengerIdx = i
			}
			if pl.Id == p.DefenderId {
				defenderIdx = i
			}
		}

		if challengerIdx == -1 || defenderIdx == -1 {
			return nil, fmt.Errorf("challenger or defender not found")
		}

		winnerIdx := -1
		loserIdx := -1
		if p.WinnerId == p.ChallengerId {
			winnerIdx = challengerIdx
			loserIdx = defenderIdx
		} else {
			winnerIdx = defenderIdx
			loserIdx = challengerIdx
		}

		// Only change rank if winner is below loser
		if winnerIdx > loserIdx {
			// Winner takes loser's position
			winner := players[winnerIdx]

			// Shift everyone from loserIdx to winnerIdx-1 down one spot
			copy(players[loserIdx+1:winnerIdx+1], players[loserIdx:winnerIdx])

			// Place winner at loser's old spot
			players[loserIdx] = winner

			// Re-assign ranks
			for i := loserIdx; i <= winnerIdx; i++ {
				players[i].Rank = int32(i + 1)
			}
		}

	case storagepb.TransactionType_INVALIDATE_MATCH:
		// Logic handled in InvalidateMatchResult by replaying
		return currentPlayers, nil
	}

	return players, nil
}

// ListPlayers returns the current player list
func (m *Model) ListPlayers() []*ladderpb.Player {
	m.mu.RLock()
	defer m.mu.RUnlock()

	players, err := m.CurrentState()
	if err != nil {
		log.Printf("Error reading current state: %v\n", err)
		return []*ladderpb.Player{}
	}
	return players
}

// AddPlayer adds a player to the ladder
func (m *Model) AddPlayer(name, playerID string) (*ladderpb.Player, error) {
	if playerID == "" {
		playerID = uuid.New().String()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Get Current State
	currentPlayers, err := m.CurrentState()
	if err != nil {
		return nil, err
	}

	// 2. Prepare Payload
	payload := &storagepb.AddPlayerStorage{
		PlayerId: playerID,
		Name:     name,
	}

	// 3. Compute New State
	newPlayers, err := m.applyTransactionLogic(storagepb.TransactionType_ADD_PLAYER, payload, currentPlayers)
	if err != nil {
		return nil, err
	}

	// 4. Create Transaction
	txID := uuid.New().String()
	timestamp := time.Now().UnixMilli()

	txProto := &storagepb.TransactionStorage{
		Id:          txID,
		Type:        storagepb.TransactionType_ADD_PLAYER,
		TimestampMs: timestamp,
		Payload:     &storagepb.TransactionStorage_AddPlayerPayload{AddPlayerPayload: payload},
		PlayerList:  ladderToStorage(newPlayers),
	}

	txBytes, err := proto.Marshal(txProto)
	if err != nil {
		return nil, err
	}

	ranksBytes, err := proto.Marshal(&storagepb.TransactionStorage{PlayerList: ladderToStorage(newPlayers)})
	if err != nil {
		return nil, err
	}

	// 5. Execute DB Transaction
	sqlTx, err := m.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer sqlTx.Rollback()

	// Insert Player
	// The new player is the last one in newPlayers
	newP := newPlayers[len(newPlayers)-1]
	if _, err := sqlTx.Exec("INSERT INTO players (id, name, rank) VALUES ($1, $2, $3)",
		newP.Id, newP.Name, newP.Rank); err != nil {
		return nil, err
	}

	// Insert Transaction
	if _, err := sqlTx.Exec(`INSERT INTO transactions (id, type, timestamp_ms, payload, player_ranks) 
		VALUES ($1, $2, $3, $4, $5)`,
		txID, int32(storagepb.TransactionType_ADD_PLAYER), timestamp, txBytes, ranksBytes); err != nil {
		return nil, err
	}

	if err := sqlTx.Commit(); err != nil {
		return nil, err
	}

	return newP, nil
}

// RemovePlayer removes a player from the ladder
func (m *Model) RemovePlayer(playerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	currentPlayers, err := m.CurrentState()
	if err != nil {
		return err
	}

	payload := &storagepb.RemovePlayerStorage{PlayerId: playerID}

	newPlayers, err := m.applyTransactionLogic(storagepb.TransactionType_REMOVE_PLAYER, payload, currentPlayers)
	if err != nil {
		return err
	}

	txID := uuid.New().String()
	timestamp := time.Now().UnixMilli()

	txProto := &storagepb.TransactionStorage{
		Id:          txID,
		Type:        storagepb.TransactionType_REMOVE_PLAYER,
		TimestampMs: timestamp,
		Payload:     &storagepb.TransactionStorage_RemovePlayerPayload{RemovePlayerPayload: payload},
		PlayerList:  ladderToStorage(newPlayers),
	}
	txBytes, err := proto.Marshal(txProto)
	if err != nil {
		return err
	}
	ranksBytes, err := proto.Marshal(&storagepb.TransactionStorage{PlayerList: ladderToStorage(newPlayers)})
	if err != nil {
		return err
	}

	sqlTx, err := m.Db.Begin()
	if err != nil {
		return err
	}
	defer sqlTx.Rollback()

	// Remove Player
	if _, err := sqlTx.Exec("DELETE FROM players WHERE id = $1", playerID); err != nil {
		return err
	}

	// Re-rank others
	for _, p := range newPlayers {
		if _, err := sqlTx.Exec("UPDATE players SET rank = $1 WHERE id = $2", p.Rank, p.Id); err != nil {
			return err
		}
	}

	// Insert Transaction
	if _, err := sqlTx.Exec(`INSERT INTO transactions (id, type, timestamp_ms, payload, player_ranks) 
		VALUES ($1, $2, $3, $4, $5)`,
		txID, int32(storagepb.TransactionType_REMOVE_PLAYER), timestamp, txBytes, ranksBytes); err != nil {
		return err
	}

	return sqlTx.Commit()
}

// AddMatchResult records a match
func (m *Model) AddMatchResult(challengerID, defenderID, winnerID string, setScores []*ladderpb.SetScore) (string, error) {
	if winnerID != challengerID && winnerID != defenderID {
		return "", fmt.Errorf("winner must be one of the players")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	currentPlayers, err := m.CurrentState()
	if err != nil {
		return "", err
	}

	storageSetScores := make([]*storagepb.SetScoreStorage, len(setScores))
	for i, s := range setScores {
		storageSetScores[i] = &storagepb.SetScoreStorage{
			ChallengerPoints:  s.ChallengerPoints,
			DefenderPoints:    s.DefenderPoints,
			ChallengerDefault: s.ChallengerDefault,
			DefenderDefault:   s.DefenderDefault,
		}
	}

	payload := &storagepb.MatchResultStorage{
		ChallengerId: challengerID,
		DefenderId:   defenderID,
		WinnerId:     winnerID,
		SetScores:    storageSetScores,
	}

	newPlayers, err := m.applyTransactionLogic(storagepb.TransactionType_MATCH_RESULT, payload, currentPlayers)
	if err != nil {
		return "", err
	}

	txID := uuid.New().String()
	timestamp := time.Now().UnixMilli()

	txProto := &storagepb.TransactionStorage{
		Id:          txID,
		Type:        storagepb.TransactionType_MATCH_RESULT,
		TimestampMs: timestamp,
		Payload:     &storagepb.TransactionStorage_MatchResultPayload{MatchResultPayload: payload},
		PlayerList:  ladderToStorage(newPlayers),
	}
	txBytes, err := proto.Marshal(txProto)
	if err != nil {
		return "", err
	}
	ranksBytes, err := proto.Marshal(&storagepb.TransactionStorage{PlayerList: ladderToStorage(newPlayers)})
	if err != nil {
		return "", err
	}

	sqlTx, err := m.Db.Begin()
	if err != nil {
		return "", err
	}
	defer sqlTx.Rollback()

	// Update Ranks in DB
	for _, p := range newPlayers {
		if _, err := sqlTx.Exec("UPDATE players SET rank = $1 WHERE id = $2", p.Rank, p.Id); err != nil {
			return "", err
		}
	}

	// Insert Transaction
	if _, err := sqlTx.Exec(`INSERT INTO transactions (id, type, timestamp_ms, payload, player_ranks) 
		VALUES ($1, $2, $3, $4, $5)`,
		txID, int32(storagepb.TransactionType_MATCH_RESULT), timestamp, txBytes, ranksBytes); err != nil {
		return "", err
	}

	if err := sqlTx.Commit(); err != nil {
		return "", err
	}

	return txID, nil
}

// InvalidateMatchResult undoes a transaction by rebuilding the state
func (m *Model) InvalidateMatchResult(txID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Get the transaction to be invalidated
	var targetTimestamp int64
	err := m.Db.QueryRow("SELECT timestamp_ms FROM transactions WHERE id = $1", txID).Scan(&targetTimestamp)
	if err == sql.ErrNoRows {
		return fmt.Errorf("transaction not found")
	}
	if err != nil {
		return err
	}

	// 2. Mark as invalidated
	sqlTx, err := m.Db.Begin()
	if err != nil {
		return err
	}
	defer sqlTx.Rollback()

	if _, err := sqlTx.Exec("UPDATE transactions SET is_invalidated = TRUE WHERE id = $1", txID); err != nil {
		return err
	}

	// 3. Find the valid snapshot *before* this transaction
	var lastSnapshotBytes []byte
	var prevTimestamp int64 = -1
	err = sqlTx.QueryRow(`
		SELECT timestamp_ms, player_ranks FROM transactions 
		WHERE timestamp_ms < $1 AND is_invalidated = FALSE
		ORDER BY timestamp_ms DESC LIMIT 1`, targetTimestamp).Scan(&prevTimestamp, &lastSnapshotBytes)

	var currentPlayers []*ladderpb.Player

	if err == sql.ErrNoRows {
		// No previous transaction => empty state
		currentPlayers = []*ladderpb.Player{}
		prevTimestamp = -1
	} else if err != nil {
		return err
	} else {
		var snapWrapper storagepb.TransactionStorage
		if err := proto.Unmarshal(lastSnapshotBytes, &snapWrapper); err != nil {
			return fmt.Errorf("failed to unmarshal snapshot: %v", err)
		}
		currentPlayers = storageToLadder(snapWrapper.PlayerList)
	}

	// 4. Fetch all SUBSEQUENT transactions
	rows, err := sqlTx.Query(`
		SELECT id, type, payload 
		FROM transactions 
		WHERE timestamp_ms > $1 AND is_invalidated = FALSE 
		ORDER BY timestamp_ms ASC`, prevTimestamp)
	if err != nil {
		return err
	}

	// Collect all txs to re-apply
	type TxInfo struct {
		Id      string
		Type    int32
		Payload []byte
	}
	var txsToReplay []TxInfo
	for rows.Next() {
		var ti TxInfo
		if err := rows.Scan(&ti.Id, &ti.Type, &ti.Payload); err != nil {
			rows.Close()
			return err
		}
		txsToReplay = append(txsToReplay, ti)
	}
	rows.Close()

	// 5. Replay
	for _, ti := range txsToReplay {
		var t storagepb.TransactionStorage
		if err := proto.Unmarshal(ti.Payload, &t); err != nil {
			return err
		}

		var payload interface{}
		switch storagepb.TransactionType(ti.Type) {
		case storagepb.TransactionType_ADD_PLAYER:
			payload = t.GetAddPlayerPayload()
		case storagepb.TransactionType_REMOVE_PLAYER:
			payload = t.GetRemovePlayerPayload()
		case storagepb.TransactionType_MATCH_RESULT:
			payload = t.GetMatchResultPayload()
		}

		newPlayers, err := m.applyTransactionLogic(storagepb.TransactionType(ti.Type), payload, currentPlayers)
		if err != nil {
			log.Printf("Consistency Error during replay tx %s: %v", ti.Id, err)
			return err // If we can't replay, we are broken.
		}
		currentPlayers = newPlayers

		// Update this transaction's snapshot in DB
		ranksBytes, _ := proto.Marshal(&storagepb.TransactionStorage{PlayerList: ladderToStorage(newPlayers)})
		if _, err := sqlTx.Exec("UPDATE transactions SET player_ranks = $1 WHERE id = $2", ranksBytes, ti.Id); err != nil {
			return err
		}
	}

	// 6. Update Players Table
	if _, err := sqlTx.Exec("DELETE FROM players"); err != nil {
		return err
	}
	for _, p := range currentPlayers {
		if _, err := sqlTx.Exec("INSERT INTO players (id, name, rank) VALUES ($1, $2, $3)", p.Id, p.Name, p.Rank); err != nil {
			return err
		}
	}

	// 7. Insert Invalidate Event
	invPayload := &storagepb.InvalidateMatchStorage{InvalidatedTransactionId: txID}
	invTxProto := &storagepb.TransactionStorage{
		Id:          uuid.New().String(),
		Type:        storagepb.TransactionType_INVALIDATE_MATCH,
		TimestampMs: time.Now().UnixMilli(),
		Payload:     &storagepb.TransactionStorage_InvalidateMatchPayload{InvalidateMatchPayload: invPayload},
		PlayerList:  ladderToStorage(currentPlayers),
	}
	invBytes, _ := proto.Marshal(invTxProto)
	invRanks, _ := proto.Marshal(&storagepb.TransactionStorage{PlayerList: ladderToStorage(currentPlayers)})

	if _, err := sqlTx.Exec(`INSERT INTO transactions (id, type, timestamp_ms, payload, player_ranks) 
		VALUES ($1, $2, $3, $4, $5)`,
		invTxProto.Id, int32(storagepb.TransactionType_INVALIDATE_MATCH), invTxProto.TimestampMs, invBytes, invRanks); err != nil {
		return err
	}

	return sqlTx.Commit()
}

// GetRecentMatches returns the last n matches
func (m *Model) GetRecentMatches(limit int32) ([]*ladderpb.MatchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rows, err := m.Db.Query(`
		SELECT payload, timestamp_ms, id 
		FROM transactions 
		WHERE type = $1 AND is_invalidated = FALSE 
		ORDER BY timestamp_ms DESC LIMIT $2`,
		storagepb.TransactionType_MATCH_RESULT, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*ladderpb.MatchResult

	for rows.Next() {
		var payloadBytes []byte
		var ts int64
		var txID string
		if err := rows.Scan(&payloadBytes, &ts, &txID); err != nil {
			return nil, err
		}

		var t storagepb.TransactionStorage
		if err := proto.Unmarshal(payloadBytes, &t); err != nil {
			continue
		}
		mr := t.GetMatchResultPayload()
		if mr == nil {
			continue
		}

		setScores := make([]*ladderpb.SetScore, len(mr.SetScores))
		for j, s := range mr.SetScores {
			setScores[j] = &ladderpb.SetScore{
				ChallengerPoints:  s.ChallengerPoints,
				DefenderPoints:    s.DefenderPoints,
				ChallengerDefault: s.ChallengerDefault,
				DefenderDefault:   s.DefenderDefault,
			}
		}

		matches = append(matches, &ladderpb.MatchResult{
			ChallengerId:  mr.ChallengerId,
			DefenderId:    mr.DefenderId,
			WinnerId:      mr.WinnerId,
			SetScores:     setScores,
			TimestampMs:   ts,
			TransactionId: txID,
		})
	}
	return matches, nil
}

// CreateInvitation generates a secure token for claiming a player
func (m *Model) CreateInvitation(playerID string) (string, error) {
	// Generate random token
	token := uuid.New().String()
	// Expires in 48 hours
	expiresAt := time.Now().Add(48 * time.Hour).UnixMilli()

	if _, err := m.Db.Exec("INSERT INTO invitations (token, player_id, expires_at) VALUES ($1, $2, $3)", token, playerID, expiresAt); err != nil {
		return "", err
	}
	return token, nil
}

// ClaimPlayer links a user to a player using a token
func (m *Model) ClaimPlayer(userID, token string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var playerID string
	var expiresAt int64

	// 1. Verify Token
	err := m.Db.QueryRow("SELECT player_id, expires_at FROM invitations WHERE token = $1", token).Scan(&playerID, &expiresAt)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("invalid invitation token")
	}
	if err != nil {
		return "", err
	}

	if time.Now().UnixMilli() > expiresAt {
		return "", fmt.Errorf("invitation expired")
	}

	// 2. Check if player is already claimed
	var existingUserID sql.NullString
	if err := m.Db.QueryRow("SELECT user_id FROM players WHERE id = $1", playerID).Scan(&existingUserID); err != nil {
		return "", err
	}
	if existingUserID.Valid && existingUserID.String != "" {
		return "", fmt.Errorf("player already claimed")
	}

	// 3. Link User
	if _, err := m.Db.Exec("UPDATE players SET user_id = $1 WHERE id = $2", userID, playerID); err != nil {
		return "", err
	}

	// 4. Delete Invitation
	if _, err := m.Db.Exec("DELETE FROM invitations WHERE token = $1", token); err != nil {
		log.Printf("Failed to delete used invitation: %v", err)
	}

	return playerID, nil
}
