type Session struct {
    UserID    int64
    ExpiresAt time.Time
}

type SessionManager struct {
    sessions map[string]Session
    mu       sync.Mutex
}

func (sm *SessionManager) CreateSession(userID int64) (string, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    sessionID, err := generateSessionID(32)
    if err != nil {
        return "", err
    }
    
    sm.sessions[sessionID] = Session{
        UserID:    userID,
        ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 giorni
    }
    return sessionID, nil
}

func (sm *SessionManager) GetUserIDBySessionID(sessionID string) (int64, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    session, exists := sm.sessions[sessionID]
    if !exists {
        return 0, errors.New("session not found")
    }

    if time.Now().After(session.ExpiresAt) {
        delete(sm.sessions, sessionID)
        return 0, errors.New("session expired")
    }

    return session.UserID, nil
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
