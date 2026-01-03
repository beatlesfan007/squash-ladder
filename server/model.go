package server

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	ladderpb "squash-ladder/server/gen/ladder"

	"github.com/google/uuid"
	"github.com/icza/backscanner"
)

// TransactionType defines the type of transaction
type TransactionType string

const (
	TxAddPlayer       TransactionType = "ADD_PLAYER"
	TxRemovePlayer    TransactionType = "REMOVE_PLAYER"
	TxMatchResult     TransactionType = "MATCH_RESULT"
	TxInvalidateMatch TransactionType = "INVALIDATE_MATCH"
)

// Transaction represents a single operation in the log
type Transaction struct {
	ID         string             `json:"id"`
	Type       TransactionType    `json:"type"`
	Timestamp  time.Time          `json:"timestamp"`
	Payload    json.RawMessage    `json:"payload"`
	PlayerList []*ladderpb.Player `json:"player_list"`
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

type InvalidateMatchPayload struct {
	InvalidatedTransactionID string `json:"invalidated_transaction_id"`
}

// Model manages the state of the squash ladder
type Model struct {
	mu          sync.RWMutex
	LogFilePath string
}

// NewModel creates a new model
func NewModel(logFilePath string) (*Model, error) {
	return &Model{
		LogFilePath: logFilePath,
	}, nil
}

// CurrentState reads the log backwards to find the last transaction and return its player list
func (m *Model) CurrentState() ([]*ladderpb.Player, error) {
	file, err := os.Open(m.LogFilePath)
	if os.IsNotExist(err) {
		return []*ladderpb.Player{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if stat.Size() == 0 {
		return []*ladderpb.Player{}, nil
	}

	scanner := backscanner.New(file, int(stat.Size()))

	// Scan backwards for the first valid line
	for {
		line, _, err := scanner.Line()
		if err != nil {
			// EOF or other error
			if err.Error() == "EOF" { // backscanner returns EOF when done
				return []*ladderpb.Player{}, nil
			}
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var lastTx Transaction
		if err := json.Unmarshal([]byte(line), &lastTx); err != nil {
			// If we can't parse the last line, maybe it's corrupted or partial?
			// We could try providing the previous line... but in a strict append log,
			// the last line should be valid.
			return nil, fmt.Errorf("failed to parse last transaction: %v", err)
		}

		if lastTx.PlayerList == nil {
			return []*ladderpb.Player{}, nil
		}
		return lastTx.PlayerList, nil
	}
}

// applyTransactionLogic calculates the NEW player state based on a transaction and previous state.
// It returns the new list of players. Not to be confused with applying to a stateful Model.
func (m *Model) applyTransactionLogic(txType TransactionType, payload json.RawMessage, currentPlayers []*ladderpb.Player) ([]*ladderpb.Player, error) {
	// Deep copy players to avoid mutating the passed slice if it's used elsewhere
	players := make([]*ladderpb.Player, len(currentPlayers))
	for i, p := range currentPlayers {
		// Create a new struct copy manually to avoid copying the mutex in MessageState
		players[i] = &ladderpb.Player{
			Id:   p.Id,
			Name: p.Name,
			Rank: p.Rank,
		}
	}

	switch txType {
	case TxAddPlayer:
		var p AddPlayerPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		// Check duplicates
		for _, pl := range players {
			if pl.Id == p.PlayerID {
				return nil, fmt.Errorf("player ID already exists")
			}
		}
		newPlayer := &ladderpb.Player{
			Id:   p.PlayerID,
			Name: p.Name,
			Rank: int32(len(players) + 1),
		}
		players = append(players, newPlayer)

	case TxRemovePlayer:
		var p RemovePlayerPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		idx := -1
		for i, pl := range players {
			if pl.Id == p.PlayerID {
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

	case TxMatchResult:
		var p MatchResultPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}

		challengerIdx := -1
		defenderIdx := -1
		for i, pl := range players {
			if pl.Id == p.ChallengerID {
				challengerIdx = i
			}
			if pl.Id == p.DefenderID {
				defenderIdx = i
			}
		}

		if challengerIdx == -1 || defenderIdx == -1 {
			// If players missing, we might ignore or error?
			// For replay safety, if a player is missing (maybe removed?), we can error or no-op.
			// Current logic enforces they exist.
			return nil, fmt.Errorf("challenger or defender not found")
		}

		winnerIdx := -1
		loserIdx := -1
		if p.WinnerID == p.ChallengerID {
			winnerIdx = challengerIdx
			loserIdx = defenderIdx
		} else {
			winnerIdx = defenderIdx
			loserIdx = challengerIdx
		}

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

	case TxInvalidateMatch:
		// Should not be applied recursively.
		// If we are applying a TxInvalidateMatch, it means we ARE replaying.
		// But wait, applyTransactionLogic is used to COMPUTE the state for a NEW transaction.
		// If the new transaction is itself an InvalidateMatch, the "payload" logic is complex:
		// It requires re-reading the whole history.
		// So this helper might not be suitable for TxInvalidateMatch logic directly
		// unless we pass the computed state in.
		// Actually, for InvalidateMatch, the state IS the result of the replay.
		// So we don't "apply" it to `currentPlayers` in the same way.
		// We'll handle InvalidateMatch separately in the public method.
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
		// In a real app, we should probably log this or return error.
		// Keeping signature allowed means returning empty or panic.
		// Assuming empty is safer for now.
		fmt.Printf("Error reading current state: %v\n", err)
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
	payload_bytes, _ := json.Marshal(AddPlayerPayload{
		PlayerID: playerID,
		Name:     name,
	})

	// 3. Compute New State
	newPlayers, err := m.applyTransactionLogic(TxAddPlayer, payload_bytes, currentPlayers)
	if err != nil {
		return nil, err
	}

	// 4. Create Transaction
	tx := &Transaction{
		ID:         uuid.New().String(),
		Type:       TxAddPlayer,
		Timestamp:  time.Now(),
		Payload:    payload_bytes,
		PlayerList: newPlayers,
	}

	// 5. Append
	if err := m.writeTransactionLocked(tx); err != nil {
		return nil, err
	}

	return newPlayers[len(newPlayers)-1], nil
}

func (m *Model) writeTransactionLocked(tx *Transaction) error {
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
	return nil
}

// RemovePlayer removes a player from the ladder
func (m *Model) RemovePlayer(playerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	currentPlayers, err := m.CurrentState()
	if err != nil {
		return err
	}

	payload_bytes, _ := json.Marshal(RemovePlayerPayload{PlayerID: playerID})

	newPlayers, err := m.applyTransactionLogic(TxRemovePlayer, payload_bytes, currentPlayers)
	if err != nil {
		return err
	}

	tx := &Transaction{
		ID:         uuid.New().String(),
		Type:       TxRemovePlayer,
		Timestamp:  time.Now(),
		Payload:    payload_bytes,
		PlayerList: newPlayers,
	}

	return m.writeTransactionLocked(tx)
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

	newPlayers, err := m.applyTransactionLogic(TxMatchResult, payload_bytes, currentPlayers)
	if err != nil {
		return "", err
	}

	tx := &Transaction{
		ID:         uuid.New().String(),
		Type:       TxMatchResult,
		Timestamp:  time.Now(),
		Payload:    payload_bytes,
		PlayerList: newPlayers,
	}

	if err := m.writeTransactionLocked(tx); err != nil {
		return "", err
	}

	return tx.ID, nil
}

// InvalidateMatchResult undoes a transaction by rebuilding the state without it
func (m *Model) InvalidateMatchResult(txID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, err := os.Open(m.LogFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	scanner := backscanner.New(file, int(stat.Size()))

	var replayStack []Transaction
	var found bool
	var currentPlayers []*ladderpb.Player

	// Scan backwards to find the target transaction
	for {
		line, _, err := scanner.Line()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var t Transaction
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			return err
		}

		if t.ID == txID {
			if t.Type != TxMatchResult {
				return fmt.Errorf("can only invalidate match results")
			}
			found = true

			// We found the target.
			// The state *before* this transaction is the PlayerList of the *previous* transaction
			// (which is the *next* transaction in our backward scan).

			// Peek the next valid line to get baseline state
			for {
				prevLine, _, err := scanner.Line()
				if err != nil {
					if err.Error() == "EOF" {
						// Found at start of log, so base state is empty
						currentPlayers = []*ladderpb.Player{}
						break
					}
					return err
				}
				prevLine = strings.TrimSpace(prevLine)
				if prevLine == "" {
					continue
				}

				var prevTx Transaction
				if err := json.Unmarshal([]byte(prevLine), &prevTx); err != nil {
					return fmt.Errorf("failed to parse previous transaction: %v", err)
				}
				currentPlayers = prevTx.PlayerList
				break
			}
			break
		}

		// If not target, push to stack to replay later
		// We push to front because we are reading backwards,
		// but we want to replay in chronological order later.
		// Actually, simpler: append, and then iterate replayStack in reverse.
		replayStack = append(replayStack, t)
	}

	if !found {
		return fmt.Errorf("transaction not found")
	}

	// 3. Replay
	// Iterate replayStack in REVERSE (oldest to newest)
	for i := len(replayStack) - 1; i >= 0; i-- {
		t := replayStack[i]
		newPlayers, err := m.applyTransactionLogic(t.Type, t.Payload, currentPlayers)
		if err != nil {
			return fmt.Errorf("replay failed at tx %s: %v", t.ID, err)
		}
		currentPlayers = newPlayers
	}

	// 4. Create Invalidate Transaction
	payload_bytes, _ := json.Marshal(InvalidateMatchPayload{
		InvalidatedTransactionID: txID,
	})

	tx := &Transaction{
		ID:         uuid.New().String(),
		Type:       TxInvalidateMatch,
		Timestamp:  time.Now(),
		Payload:    payload_bytes,
		PlayerList: currentPlayers,
	}

	return m.writeTransactionLocked(tx)
}

// GetRecentMatches returns the last n matches
func (m *Model) GetRecentMatches(limit int32) ([]*ladderpb.MatchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	file, err := os.Open(m.LogFilePath)
	if os.IsNotExist(err) {
		return []*ladderpb.MatchResult{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	scanner := backscanner.New(file, int(stat.Size()))

	var matches []*ladderpb.MatchResult
	invalidatedIds := make(map[string]bool)
	count := int32(0)

	for {
		if count >= limit {
			break
		}

		line, _, err := scanner.Line()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var t Transaction
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			continue
		}

		if t.Type == TxInvalidateMatch {
			var p InvalidateMatchPayload
			if err := json.Unmarshal(t.Payload, &p); err == nil {
				invalidatedIds[p.InvalidatedTransactionID] = true
			}
		} else if t.Type == TxMatchResult {
			if invalidatedIds[t.ID] {
				continue // Skip invalidated matches
			}

			var p MatchResultPayload
			if err := json.Unmarshal(t.Payload, &p); err != nil {
				continue
			}

			setScores := make([]*ladderpb.SetScore, len(p.SetScores))
			for j, s := range p.SetScores {
				setScores[j] = &ladderpb.SetScore{
					ChallengerPoints:  s.ChallengerPoints,
					DefenderPoints:    s.DefenderPoints,
					ChallengerDefault: s.ChallengerDefault,
					DefenderDefault:   s.DefenderDefault,
				}
			}

			matches = append(matches, &ladderpb.MatchResult{
				ChallengerId:  p.ChallengerID,
				DefenderId:    p.DefenderID,
				WinnerId:      p.WinnerID,
				SetScores:     setScores,
				TimestampMs:   t.Timestamp.UnixMilli(),
				TransactionId: t.ID,
			})
			count++
		}
	}

	return matches, nil
}
