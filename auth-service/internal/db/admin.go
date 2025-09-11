// auth-service/internal/db/admin.go
package db

import (
	"fmt"
	"time"
)

// AdminUserInfo rappresenta le informazioni di un utente per il pannello admin
type AdminUserInfo struct {
	ID                int64     `json:"id"`
	Username          string    `json:"username"`
	Nome              string    `json:"nome"`
	Cognome           string    `json:"cognome"`
	Email             string    `json:"email"`
	DataRegistrazione time.Time `json:"dataRegistrazione"`
	IsActive          bool      `json:"isActive"`
	IsAdmin           bool      `json:"isAdmin"`
	PostCreati        int       `json:"postCreati"`
	CommentiScritti   int       `json:"commentiScritti"`
}

// GetAllUsers restituisce tutti gli utenti per il pannello admin
func (db *Database) GetAllUsers() ([]AdminUserInfo, error) {
	query := `
		SELECT 
			u.id,
			u.username,
			u.nome,
			u.cognome,
			u.email,
			COALESCE(u.created_at, CURRENT_TIMESTAMP) as data_registrazione,
			COALESCE(u.is_active, true) as is_active,
			COALESCE(u.is_admin, false) as is_admin
		FROM users u
		ORDER BY u.created_at DESC`

	rows, err := db.Conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("errore nel recupero utenti: %v", err)
	}
	defer rows.Close()

	var users []AdminUserInfo
	for rows.Next() {
		var user AdminUserInfo
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Nome,
			&user.Cognome,
			&user.Email,
			&user.DataRegistrazione,
			&user.IsActive,
			&user.IsAdmin,
		)
		if err != nil {
			return nil, fmt.Errorf("errore nello scan utente: %v", err)
		}

		// Conta i post e commenti (chiamate separate al Python backend)
		// Per ora impostiamo valori di default
		user.PostCreati = 0
		user.CommentiScritti = 0

		users = append(users, user)
	}

	return users, nil
}

// ToggleUserStatus attiva/disattiva un utente
func (db *Database) ToggleUserStatus(userID int64) (bool, error) {
	// Prima ottieni lo status attuale
	var currentStatus bool
	err := db.Conn.QueryRow(
		"SELECT COALESCE(is_active, true) FROM users WHERE id = $1",
		userID,
	).Scan(&currentStatus)

	if err != nil {
		return false, fmt.Errorf("utente non trovato: %v", err)
	}

	// Aggiorna lo status
	newStatus := !currentStatus
	_, err = db.Conn.Exec(
		"UPDATE users SET is_active = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		newStatus,
		userID,
	)

	if err != nil {
		return false, fmt.Errorf("errore nell'aggiornamento status utente: %v", err)
	}

	fmt.Printf("[DB] ✅ Status utente %d cambiato da %t a %t\n", userID, currentStatus, newStatus)
	return newStatus, nil
}

// GetTotalUsersCount restituisce il numero totale di utenti
func (db *Database) GetTotalUsersCount() (int, error) {
	var count int
	err := db.Conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("errore nel conteggio utenti: %v", err)
	}
	return count, nil
}

// CreateUsersTableWithAdminFields aggiunge i campi necessari per l'admin se non esistono
func (db *Database) CreateUsersTableWithAdminFields() error {
	// Aggiungi colonne se non esistono
	alterQueries := []string{
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
	}

	for _, query := range alterQueries {
		_, err := db.Conn.Exec(query)
		if err != nil {
			return fmt.Errorf("errore nell'aggiornamento tabella users: %v", err)
		}
	}

	// Aggiorna tutti gli utenti esistenti che non hanno created_at
	_, err := db.Conn.Exec(`
		UPDATE users 
		SET created_at = CURRENT_TIMESTAMP 
		WHERE created_at IS NULL
	`)
	if err != nil {
		return fmt.Errorf("errore nell'aggiornamento created_at: %v", err)
	}

	fmt.Println("✔ Tabella users aggiornata con campi admin")
	return nil
}

// GetUserStats restituisce statistiche dettagliate di un utente
func (db *Database) GetUserStats(userID int64) (map[string]interface{}, error) {
	var stats map[string]interface{} = make(map[string]interface{})

	// Statistiche di base
	query := `
		SELECT 
			u.username,
			u.nome,
			u.cognome,
			u.email,
			u.created_at,
			COALESCE(u.is_active, true) as is_active,
			COALESCE(u.is_admin, false) as is_admin
		FROM users u 
		WHERE u.id = $1`

	var username, nome, cognome, email string
	var createdAt time.Time
	var isActive, isAdmin bool

	err := db.Conn.QueryRow(query, userID).Scan(
		&username, &nome, &cognome, &email, &createdAt, &isActive, &isAdmin,
	)
	if err != nil {
		return nil, fmt.Errorf("utente non trovato: %v", err)
	}

	stats["username"] = username
	stats["nome"] = nome
	stats["cognome"] = cognome
	stats["email"] = email
	stats["created_at"] = createdAt
	stats["is_active"] = isActive
	stats["is_admin"] = isAdmin

	// Conta partecipazioni eventi
	var eventCount int
	err = db.Conn.QueryRow(
		"SELECT COUNT(*) FROM event_participants WHERE user_id = $1",
		userID,
	).Scan(&eventCount)
	if err == nil {
		stats["events_participated"] = eventCount
	}

	// Conta preferiti
	var favoritesCount int
	err = db.Conn.QueryRow(
		"SELECT COUNT(*) FROM user_favorites WHERE user_id = $1",
		userID,
	).Scan(&favoritesCount)
	if err == nil {
		stats["favorites_count"] = favoritesCount
	}

	// Conta amici
	var friendsCount int
	err = db.Conn.QueryRow(
		"SELECT COUNT(*) FROM friendships WHERE user1_id = $1 OR user2_id = $1",
		userID,
	).Scan(&friendsCount)
	if err == nil {
		stats["friends_count"] = friendsCount
	}

	return stats, nil
}

// Metodi per la verifica admin esistenti (già implementati nel codice precedente)

// CheckUserIsAdmin verifica se un utente è amministratore
func (db *Database) CheckUserIsAdmin(userID int64) (bool, error) {
	var isAdmin bool
	err := db.Conn.QueryRow("SELECT COALESCE(is_admin, false) FROM users WHERE id = $1", userID).Scan(&isAdmin)
	if err != nil {
		return false, err
	}
	return isAdmin, nil
}

// IsUserActive verifica se un utente è attivo
func (db *Database) IsUserActive(userID int64) (bool, error) {
	var isActive bool
	err := db.Conn.QueryRow("SELECT COALESCE(is_active, true) FROM users WHERE id = $1", userID).Scan(&isActive)
	if err != nil {
		return false, err
	}
	return isActive, nil
}

// GetUserByEmail trova un utente tramite email (per admin)
func (db *Database) GetUserByEmail(email string) (*AdminUserInfo, error) {
	query := `
		SELECT 
			u.id,
			u.username,
			u.nome,
			u.cognome,
			u.email,
			COALESCE(u.created_at, CURRENT_TIMESTAMP) as data_registrazione,
			COALESCE(u.is_active, true) as is_active,
			COALESCE(u.is_admin, false) as is_admin
		FROM users u
		WHERE u.email = $1`

	var user AdminUserInfo
	err := db.Conn.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Nome,
		&user.Cognome,
		&user.Email,
		&user.DataRegistrazione,
		&user.IsActive,
		&user.IsAdmin,
	)

	if err != nil {
		return nil, fmt.Errorf("utente non trovato: %v", err)
	}

	return &user, nil
}