package sessions

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

type SessionManager struct {
	sessions map[string]int64
	mu       sync.Mutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]int64),
	}
}

func (sm *SessionManager) CreateSession(userID int64) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionID, err := generateSessionID(32)
	if err != nil {
		return "", err
	}
	sm.sessions[sessionID] = userID
	return sessionID, nil
}

func (sm *SessionManager) GetUserIDBySessionID(sessionID string) (int64, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	userID, exists := sm.sessions[sessionID]
	if !exists {
		return 0, errors.New("session not found")
	}
	return userID, nil
}

// Genera un ID sessione casuale
func generateSessionID(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", errors.New("impossibile generare un ID di sessione sicuro")
	}
	return hex.EncodeToString(bytes), nil
}

// In sessions/session.go, aggiungi:
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
}
