package server

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	ladderpb "squash-ladder/server/gen/ladder"
	storagepb "squash-ladder/server/gen/storage"

	"github.com/google/uuid"
	"github.com/icza/backscanner"
	"google.golang.org/protobuf/proto"
)

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

		// Decode Base64
		data, err := base64.StdEncoding.DecodeString(line)
		if err != nil {
			// If we can't decode, maybe it's corrupted or old format?
			// We treat it as error for now.
			return nil, fmt.Errorf("failed to decode line: %v", err)
		}

		var lastTx storagepb.TransactionStorage
		if err := proto.Unmarshal(data, &lastTx); err != nil {
			return nil, fmt.Errorf("failed to unmarshal last transaction: %v", err)
		}

		return storageToLadder(lastTx.PlayerList), nil
	}
}

// applyTransactionLogic calculates the NEW player state based on a transaction type and payload.
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
		// We don't apply logic on top of current state for invalidation
		// because invalidation requires replay.
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
	tx := &storagepb.TransactionStorage{
		Id:          uuid.New().String(),
		Type:        storagepb.TransactionType_ADD_PLAYER,
		TimestampMs: time.Now().UnixMilli(),
		Payload:     &storagepb.TransactionStorage_AddPlayerPayload{AddPlayerPayload: payload},
		PlayerList:  ladderToStorage(newPlayers),
	}

	// 5. Append
	if err := m.writeTransactionLocked(tx); err != nil {
		return nil, err
	}

	return newPlayers[len(newPlayers)-1], nil
}

func (m *Model) writeTransactionLocked(tx *storagepb.TransactionStorage) error {
	file, err := os.OpenFile(m.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := proto.Marshal(tx)
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	if _, err := file.WriteString(encoded + "\n"); err != nil {
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

	payload := &storagepb.RemovePlayerStorage{PlayerId: playerID}

	newPlayers, err := m.applyTransactionLogic(storagepb.TransactionType_REMOVE_PLAYER, payload, currentPlayers)
	if err != nil {
		return err
	}

	tx := &storagepb.TransactionStorage{
		Id:          uuid.New().String(),
		Type:        storagepb.TransactionType_REMOVE_PLAYER,
		TimestampMs: time.Now().UnixMilli(),
		Payload:     &storagepb.TransactionStorage_RemovePlayerPayload{RemovePlayerPayload: payload},
		PlayerList:  ladderToStorage(newPlayers),
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

	tx := &storagepb.TransactionStorage{
		Id:          uuid.New().String(),
		Type:        storagepb.TransactionType_MATCH_RESULT,
		TimestampMs: time.Now().UnixMilli(),
		Payload:     &storagepb.TransactionStorage_MatchResultPayload{MatchResultPayload: payload},
		PlayerList:  ladderToStorage(newPlayers),
	}

	if err := m.writeTransactionLocked(tx); err != nil {
		return "", err
	}

	return tx.Id, nil
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

	var replayStack []*storagepb.TransactionStorage
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

		data, err := base64.StdEncoding.DecodeString(line)
		if err != nil {
			return err
		}

		var t storagepb.TransactionStorage
		if err := proto.Unmarshal(data, &t); err != nil {
			return err
		}

		if t.Id == txID {
			if t.Type != storagepb.TransactionType_MATCH_RESULT {
				return fmt.Errorf("can only invalidate match results")
			}
			found = true

			// The player state *before* this transaction is the list from the previous transaction
			// (which is next in backward scan)
			for {
				prevLine, _, err := scanner.Line()
				if err != nil {
					if err.Error() == "EOF" {
						// Start of log
						currentPlayers = []*ladderpb.Player{}
						break
					}
					return err
				}
				prevLine = strings.TrimSpace(prevLine)
				if prevLine == "" {
					continue
				}

				prevData, err := base64.StdEncoding.DecodeString(prevLine)
				if err != nil {
					return err
				}
				var prevTx storagepb.TransactionStorage
				if err := proto.Unmarshal(prevData, &prevTx); err != nil {
					return err
				}
				currentPlayers = storageToLadder(prevTx.PlayerList)
				break
			}
			break
		}

		replayStack = append(replayStack, &t)
	}

	if !found {
		return fmt.Errorf("transaction not found")
	}

	// 3. Replay (reverse of replayStack)
	for i := len(replayStack) - 1; i >= 0; i-- {
		t := replayStack[i]

		// Extract payload interface based on oneof
		var payload interface{}
		switch t.Type {
		case storagepb.TransactionType_ADD_PLAYER:
			payload = t.GetAddPlayerPayload()
		case storagepb.TransactionType_REMOVE_PLAYER:
			payload = t.GetRemovePlayerPayload()
		case storagepb.TransactionType_MATCH_RESULT:
			payload = t.GetMatchResultPayload()
		case storagepb.TransactionType_INVALIDATE_MATCH:
			// No state change logic for this, just pass through
			continue
		}

		newPlayers, err := m.applyTransactionLogic(t.Type, payload, currentPlayers)
		if err != nil {
			return fmt.Errorf("replay failed at tx %s: %v", t.Id, err)
		}
		currentPlayers = newPlayers
	}

	// 4. Create Invalidate Transaction
	payload := &storagepb.InvalidateMatchStorage{
		InvalidatedTransactionId: txID,
	}

	tx := &storagepb.TransactionStorage{
		Id:          uuid.New().String(),
		Type:        storagepb.TransactionType_INVALIDATE_MATCH,
		TimestampMs: time.Now().UnixMilli(),
		Payload:     &storagepb.TransactionStorage_InvalidateMatchPayload{InvalidateMatchPayload: payload},
		PlayerList:  ladderToStorage(currentPlayers),
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

		data, err := base64.StdEncoding.DecodeString(line)
		if err != nil {
			continue // skip bad lines
		}

		var t storagepb.TransactionStorage
		if err := proto.Unmarshal(data, &t); err != nil {
			continue
		}

		if t.Type == storagepb.TransactionType_INVALIDATE_MATCH {
			inv := t.GetInvalidateMatchPayload()
			if inv != nil {
				invalidatedIds[inv.InvalidatedTransactionId] = true
			}
		} else if t.Type == storagepb.TransactionType_MATCH_RESULT {
			if invalidatedIds[t.Id] {
				continue // Skip invalidated matches
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
				TimestampMs:   t.TimestampMs,
				TransactionId: t.Id,
			})
			count++
		}
	}

	return matches, nil
}
