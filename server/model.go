package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	ladderpb "squash-ladder/server/gen/ladder"

	"github.com/google/uuid"
)

// TransactionType defines the type of transaction
type TransactionType string

const (
	TxAddPlayer    TransactionType = "ADD_PLAYER"
	TxRemovePlayer TransactionType = "REMOVE_PLAYER"
	TxMatchResult  TransactionType = "MATCH_RESULT"
)

// Transaction represents a single operation in the log
type Transaction struct {
	ID        string          `json:"id"`
	Type      TransactionType `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// AddPlayerPayload payload for adding a player
type AddPlayerPayload struct {
	PlayerID string `json:"player_id"`
	Name     string `json:"name"`
}

// RemovePlayerPayload payload for removing a player
type RemovePlayerPayload struct {
	PlayerID string `json:"player_id"`
}

// MatchResultPayload payload for a match result
// SetScorePayload payload for a single set
type SetScorePayload struct {
	ChallengerPoints  int32 `json:"challenger_points"`
	DefenderPoints    int32 `json:"defender_points"`
	ChallengerDefault bool  `json:"challenger_default,omitempty"`
	DefenderDefault   bool  `json:"defender_default,omitempty"`
}

// MatchResultPayload payload for a match result
type MatchResultPayload struct {
	ChallengerID string            `json:"challenger_id"`
	DefenderID   string            `json:"defender_id"`
	WinnerID     string            `json:"winner_id"`
	SetScores    []SetScorePayload `json:"set_scores"`
}

// Model manages the state of the squash ladder
type Model struct {
	mu           sync.RWMutex
	Players      []*ladderpb.Player
	LogFilePath  string
	Transactions []Transaction // In-memory cache of transactions for invalidation
}

// NewModel creates a new model and loads state from operations log
func NewModel(logFilePath string) (*Model, error) {
	m := &Model{
		Players:      make([]*ladderpb.Player, 0),
		LogFilePath:  logFilePath,
		Transactions: make([]Transaction, 0),
	}

	if err := m.loadState(); err != nil {
		return nil, err
	}

	return m, nil
}

// loadState reads the transaction log and rebuilds the ladder state
func (m *Model) loadState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Reset state
	m.Players = make([]*ladderpb.Player, 0)
	m.Transactions = make([]Transaction, 0)

	file, err := os.Open(m.LogFilePath)
	if os.IsNotExist(err) {
		return nil // New ladder
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var tx Transaction
		if err := json.Unmarshal(scanner.Bytes(), &tx); err != nil {
			return fmt.Errorf("failed to parse transaction: %v", err)
		}
		// Apply transaction to state
		if err := m.applyTransaction(&tx); err != nil {
			return fmt.Errorf("failed to apply transaction %s: %v", tx.ID, err)
		}
		m.Transactions = append(m.Transactions, tx)
	}

	return scanner.Err()
}

// applyTransaction applies a single transaction to the in-memory state
func (m *Model) applyTransaction(tx *Transaction) error {
	switch tx.Type {
	case TxAddPlayer:
		var p AddPlayerPayload
		if err := json.Unmarshal(tx.Payload, &p); err != nil {
			return err
		}
		player := &ladderpb.Player{
			Id:   p.PlayerID,
			Name: p.Name,
			Rank: int32(len(m.Players) + 1),
		}
		m.Players = append(m.Players, player)

	case TxRemovePlayer:
		var p RemovePlayerPayload
		if err := json.Unmarshal(tx.Payload, &p); err != nil {
			return err
		}
		m.removePlayerInternal(p.PlayerID)

	case TxMatchResult:
		var p MatchResultPayload
		if err := json.Unmarshal(tx.Payload, &p); err != nil {
			return err
		}
		m.applyMatchResultInternal(p.ChallengerID, p.DefenderID, p.WinnerID)
	}
	return nil
}

func (m *Model) removePlayerInternal(playerID string) {
	idx := -1
	for i, p := range m.Players {
		if p.Id == playerID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return // Player not found
	}

	// Remove player
	m.Players = append(m.Players[:idx], m.Players[idx+1:]...)

	// Update ranks for subsequent players
	for i := idx; i < len(m.Players); i++ {
		m.Players[i].Rank = int32(i + 1)
	}
}

func (m *Model) applyMatchResultInternal(challengerID, defenderID, winnerID string) {
	challengerIdx := -1
	defenderIdx := -1

	for i, p := range m.Players {
		if p.Id == challengerID {
			challengerIdx = i
		}
		if p.Id == defenderID {
			defenderIdx = i
		}
	}

	if challengerIdx == -1 || defenderIdx == -1 {
		return
	}

	winnerIdx := -1
	loserIdx := -1

	if winnerID == challengerID {
		winnerIdx = challengerIdx
		loserIdx = defenderIdx
	} else {
		winnerIdx = defenderIdx
		loserIdx = challengerIdx
	}

	// If winner is lower rank (higher index value), they take loser's spot
	// Lower rank number = better rank
	// Array index 0 = Rank 1
	// So if winnerIdx > loserIdx, winner has worse rank and beat better rank
	if winnerIdx > loserIdx {
		// Winner takes loser's position
		winner := m.Players[winnerIdx]

		// Shift everyone from loserIdx to winnerIdx-1 down one spot
		copy(m.Players[loserIdx+1:winnerIdx+1], m.Players[loserIdx:winnerIdx])

		// Place winner at loser's old spot
		m.Players[loserIdx] = winner

		// Re-assign ranks
		for i := loserIdx; i <= winnerIdx; i++ {
			m.Players[i].Rank = int32(i + 1)
		}
	}
}

// ListPlayers returns a copy of the current player list
func (m *Model) ListPlayers() []*ladderpb.Player {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return copy to prevent race conditions
	result := make([]*ladderpb.Player, len(m.Players))
	copy(result, m.Players)
	return result
}

// appendTransaction writes a transaction to the log and updates state
func (m *Model) appendTransaction(tx *Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Write to file
	file, err := os.OpenFile(m.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}

	// Apply to local state
	if err := m.applyTransaction(tx); err != nil {
		return fmt.Errorf("failed to apply transaction: %v", err)
	}

	m.Transactions = append(m.Transactions, *tx)
	return nil
}

// AddPlayer adds a player to the ladder
func (m *Model) AddPlayer(name, playerID string) (*ladderpb.Player, error) {
	if playerID == "" {
		playerID = uuid.New().String()
	}

	// Check if ID exists
	m.mu.RLock()
	for _, p := range m.Players {
		if p.Id == playerID {
			m.mu.RUnlock()
			return nil, fmt.Errorf("player ID already exists")
		}
	}
	m.mu.RUnlock()

	payload_bytes, _ := json.Marshal(AddPlayerPayload{
		PlayerID: playerID,
		Name:     name,
	})

	tx := &Transaction{
		ID:        uuid.New().String(),
		Type:      TxAddPlayer,
		Timestamp: time.Now(),
		Payload:   payload_bytes,
	}

	if err := m.appendTransaction(tx); err != nil {
		return nil, err
	}

	// Get the added player
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Players[len(m.Players)-1], nil
}

// RemovePlayer removes a player from the ladder
func (m *Model) RemovePlayer(playerID string) error {
	m.mu.RLock()
	found := false
	for _, p := range m.Players {
		if p.Id == playerID {
			found = true
			break
		}
	}
	m.mu.RUnlock()

	if !found {
		return fmt.Errorf("player not found")
	}

	payload_bytes, _ := json.Marshal(RemovePlayerPayload{PlayerID: playerID})
	tx := &Transaction{
		ID:        uuid.New().String(),
		Type:      TxRemovePlayer,
		Timestamp: time.Now(),
		Payload:   payload_bytes,
	}

	return m.appendTransaction(tx)
}

// AddMatchResult records a match
func (m *Model) AddMatchResult(challengerID, defenderID, winnerID string, setScores []*ladderpb.SetScore) (string, error) {
	// Note: Validation logic moved to service layer

	if winnerID != challengerID && winnerID != defenderID {
		return "", fmt.Errorf("winner must be one of the players")
	}

	payloadSets := make([]SetScorePayload, len(setScores))
	for i, s := range setScores {
		payloadSets[i] = SetScorePayload{
			ChallengerPoints:  s.ChallengerPoints,
			DefenderPoints:    s.DefenderPoints,
			ChallengerDefault: s.ChallengerDefault,
			DefenderDefault:   s.DefenderDefault,
		}
	}

	payload_bytes, _ := json.Marshal(MatchResultPayload{
		ChallengerID: challengerID,
		DefenderID:   defenderID,
		WinnerID:     winnerID,
		SetScores:    payloadSets,
	})

	tx := &Transaction{
		ID:        uuid.New().String(),
		Type:      TxMatchResult,
		Timestamp: time.Now(),
		Payload:   payload_bytes,
	}

	if err := m.appendTransaction(tx); err != nil {
		return "", err
	}

	return tx.ID, nil
}

// InvalidateMatchResult undoes a transaction by rebuilding the state without it
func (m *Model) InvalidateMatchResult(txID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Find the transaction to invalidate
	txIdx := -1
	for i, tx := range m.Transactions {
		if tx.ID == txID {
			txIdx = i
			break
		}
	}

	if txIdx == -1 {
		return fmt.Errorf("transaction not found")
	}

	// 2. Remove from In-Memory list
	m.Transactions = append(m.Transactions[:txIdx], m.Transactions[txIdx+1:]...)

	// 3. Rewrite the entire log file
	file, err := os.OpenFile(m.LogFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, tx := range m.Transactions {
		data, err := json.Marshal(tx)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return err
		}
	}

	// 4. Rebuild State from scratch (in-memory) to ensure correctness
	// We can reuse logic by resetting players and re-applying all transactions
	m.Players = make([]*ladderpb.Player, 0)
	for _, tx := range m.Transactions {
		if err := m.applyTransaction(&tx); err != nil {
			// This shouldn't happen if they were valid before
			return fmt.Errorf("critical error rebuilding state: %v", err)
		}
	}

	return nil
}

// GetRecentMatches returns the last n matches
func (m *Model) GetRecentMatches(limit int32) ([]*ladderpb.MatchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]*ladderpb.MatchResult, 0)
	count := int32(0)

	// Iterate backwards through transactions
	for i := len(m.Transactions) - 1; i >= 0; i-- {
		if count >= limit {
			break
		}
		tx := m.Transactions[i]
		if tx.Type == TxMatchResult {
			var p MatchResultPayload
			if err := json.Unmarshal(tx.Payload, &p); err != nil {
				return nil, fmt.Errorf("failed to parse match result: %v", err)
			}

			// Convert payload to proto
			setScores := make([]*ladderpb.SetScore, len(p.SetScores))
			for j, s := range p.SetScores {
				setScores[j] = &ladderpb.SetScore{
					ChallengerPoints:  s.ChallengerPoints,
					DefenderPoints:    s.DefenderPoints,
					ChallengerDefault: s.ChallengerDefault,
					DefenderDefault:   s.DefenderDefault,
				}
			}

			results = append(results, &ladderpb.MatchResult{
				ChallengerId:  p.ChallengerID,
				DefenderId:    p.DefenderID,
				WinnerId:      p.WinnerID,
				SetScores:     setScores,
				TimestampMs:   tx.Timestamp.UnixMilli(),
				TransactionId: tx.ID,
			})
			count++
		}
	}
	return results, nil
}
