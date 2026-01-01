package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	playerspb "squash-ladder/server/gen/players"

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
type MatchResultPayload struct {
	Player1ID string   `json:"player1_id"`
	Player2ID string   `json:"player2_id"`
	WinnerID  string   `json:"winner_id"`
	SetScores []string `json:"set_scores"`
}

// Ladder manages the state of the squash ladder
type Ladder struct {
	mu           sync.RWMutex
	Players      []*playerspb.Player
	LogFilePath  string
	Transactions []Transaction // In-memory cache of transactions for invalidation
}

// NewLadder creates a new ladder and loads state from operations log
func NewLadder(logFilePath string) (*Ladder, error) {
	l := &Ladder{
		Players:      make([]*playerspb.Player, 0),
		LogFilePath:  logFilePath,
		Transactions: make([]Transaction, 0),
	}

	if err := l.loadState(); err != nil {
		return nil, err
	}

	return l, nil
}

// loadState reads the transaction log and rebuilds the ladder state
func (l *Ladder) loadState() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Reset state
	l.Players = make([]*playerspb.Player, 0)
	l.Transactions = make([]Transaction, 0)

	file, err := os.Open(l.LogFilePath)
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
		if err := l.applyTransaction(&tx); err != nil {
			return fmt.Errorf("failed to apply transaction %s: %v", tx.ID, err)
		}
		l.Transactions = append(l.Transactions, tx)
	}

	return scanner.Err()
}

// applyTransaction applies a single transaction to the in-memory state
func (l *Ladder) applyTransaction(tx *Transaction) error {
	switch tx.Type {
	case TxAddPlayer:
		var p AddPlayerPayload
		if err := json.Unmarshal(tx.Payload, &p); err != nil {
			return err
		}
		player := &playerspb.Player{
			Id:   p.PlayerID,
			Name: p.Name,
			Rank: int32(len(l.Players) + 1),
		}
		l.Players = append(l.Players, player)

	case TxRemovePlayer:
		var p RemovePlayerPayload
		if err := json.Unmarshal(tx.Payload, &p); err != nil {
			return err
		}
		l.removePlayerInternal(p.PlayerID)

	case TxMatchResult:
		var p MatchResultPayload
		if err := json.Unmarshal(tx.Payload, &p); err != nil {
			return err
		}
		l.applyMatchResultInternal(p.Player1ID, p.Player2ID, p.WinnerID)
	}
	return nil
}

func (l *Ladder) removePlayerInternal(playerID string) {
	idx := -1
	for i, p := range l.Players {
		if p.Id == playerID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return // Player not found
	}

	// Remove player
	l.Players = append(l.Players[:idx], l.Players[idx+1:]...)

	// Update ranks for subsequent players
	for i := idx; i < len(l.Players); i++ {
		l.Players[i].Rank = int32(i + 1)
	}
}

func (l *Ladder) applyMatchResultInternal(p1ID, p2ID, winnerID string) {
	p1Idx := -1
	p2Idx := -1

	for i, p := range l.Players {
		if p.Id == p1ID {
			p1Idx = i
		}
		if p.Id == p2ID {
			p2Idx = i
		}
	}

	if p1Idx == -1 || p2Idx == -1 {
		return
	}

	winnerIdx := -1
	loserIdx := -1

	if winnerID == p1ID {
		winnerIdx = p1Idx
		loserIdx = p2Idx
	} else {
		winnerIdx = p2Idx
		loserIdx = p1Idx
	}

	// If winner is lower rank (higher index value), they take loser's spot
	// Lower rank number = better rank
	// Array index 0 = Rank 1
	// So if winnerIdx > loserIdx, winner has worse rank and beat better rank
	if winnerIdx > loserIdx {
		// Winner takes loser's position
		winner := l.Players[winnerIdx]

		// Shift everyone from loserIdx to winnerIdx-1 down one spot
		copy(l.Players[loserIdx+1:winnerIdx+1], l.Players[loserIdx:winnerIdx])

		// Place winner at loser's old spot
		l.Players[loserIdx] = winner

		// Re-assign ranks
		for i := loserIdx; i <= winnerIdx; i++ {
			l.Players[i].Rank = int32(i + 1)
		}
	}
}

// ListPlayers returns a copy of the current player list
func (l *Ladder) ListPlayers() []*playerspb.Player {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return copy to prevent race conditions
	result := make([]*playerspb.Player, len(l.Players))
	copy(result, l.Players)
	return result
}

// appendTransaction writes a transaction to the log and updates state
func (l *Ladder) appendTransaction(tx *Transaction) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Write to file
	file, err := os.OpenFile(l.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	if err := l.applyTransaction(tx); err != nil {
		return fmt.Errorf("failed to apply transaction: %v", err)
	}

	l.Transactions = append(l.Transactions, *tx)
	return nil
}

// AddPlayer adds a player to the ladder
func (l *Ladder) AddPlayer(name, playerID string) (*playerspb.Player, error) {
	if playerID == "" {
		playerID = uuid.New().String()
	}

	// Check if ID exists
	l.mu.RLock()
	for _, p := range l.Players {
		if p.Id == playerID {
			l.mu.RUnlock()
			return nil, fmt.Errorf("player ID already exists")
		}
	}
	l.mu.RUnlock()

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

	if err := l.appendTransaction(tx); err != nil {
		return nil, err
	}

	// Get the added player
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.Players[len(l.Players)-1], nil
}

// RemovePlayer removes a player from the ladder
func (l *Ladder) RemovePlayer(playerID string) error {
	l.mu.RLock()
	found := false
	for _, p := range l.Players {
		if p.Id == playerID {
			found = true
			break
		}
	}
	l.mu.RUnlock()

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

	return l.appendTransaction(tx)
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

// AddMatchResult records a match
func (l *Ladder) AddMatchResult(p1ID, p2ID, winnerID string, setScores []string) (string, error) {
	// Validate
	if valid, err := ValidateScore(setScores); !valid {
		return "", err
	}

	if winnerID != p1ID && winnerID != p2ID {
		return "", fmt.Errorf("winner must be one of the players")
	}

	payload_bytes, _ := json.Marshal(MatchResultPayload{
		Player1ID: p1ID,
		Player2ID: p2ID,
		WinnerID:  winnerID,
		SetScores: setScores,
	})

	tx := &Transaction{
		ID:        uuid.New().String(),
		Type:      TxMatchResult,
		Timestamp: time.Now(),
		Payload:   payload_bytes,
	}

	if err := l.appendTransaction(tx); err != nil {
		return "", err
	}

	return tx.ID, nil
}

// InvalidateMatchResult undoes a transaction by rebuilding the state without it
// Note: This is an expensive operation as implemented (re-reading log),
// but meets the requirements for now.
func (l *Ladder) InvalidateMatchResult(txID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 1. Find the transaction to invalidate
	txIdx := -1
	for i, tx := range l.Transactions {
		if tx.ID == txID {
			txIdx = i
			break
		}
	}

	if txIdx == -1 {
		return fmt.Errorf("transaction not found")
	}

	// 2. Remove from In-Memory list
	l.Transactions = append(l.Transactions[:txIdx], l.Transactions[txIdx+1:]...)

	// 3. Rewrite the entire log file
	file, err := os.OpenFile(l.LogFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, tx := range l.Transactions {
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
	l.Players = make([]*playerspb.Player, 0)
	for _, tx := range l.Transactions {
		if err := l.applyTransaction(&tx); err != nil {
			// This shouldn't happen if they were valid before
			return fmt.Errorf("critical error rebuilding state: %v", err)
		}
	}

	return nil
}
