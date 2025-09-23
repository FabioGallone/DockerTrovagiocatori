package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"trovagiocatoriAuth/internal/models"
)

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// GetAllUsers restituisce tutti gli utenti per il pannello admin
func (r *AdminRepository) GetAllUsers() ([]models.AdminUserInfo, error) {
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

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("errore nel recupero utenti: %v", err)
	}
	defer rows.Close()

	var users []models.AdminUserInfo
	for rows.Next() {
		var user models.AdminUserInfo
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

		//valori di default per post e commenti
		user.PostCreati = 0
		user.CommentiScritti = 0

		users = append(users, user)
	}

	return users, nil
}

// ToggleUserStatus attiva/disattiva un utente
func (r *AdminRepository) ToggleUserStatus(userID int64) (bool, error) {
	// Prima ottieni lo status attuale
	var currentStatus bool
	err := r.db.QueryRow(
		"SELECT COALESCE(is_active, true) FROM users WHERE id = $1",
		userID,
	).Scan(&currentStatus)

	if err != nil {
		return false, fmt.Errorf("utente non trovato: %v", err)
	}

	// Aggiorna lo status
	newStatus := !currentStatus
	_, err = r.db.Exec(
		"UPDATE users SET is_active = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		newStatus,
		userID,
	)

	if err != nil {
		return false, fmt.Errorf("errore nell'aggiornamento status utente: %v", err)
	}

	fmt.Printf("[DB] Status utente %d cambiato da %t a %t\n", userID, currentStatus, newStatus)
	return newStatus, nil
}

// GetTotalUsersCount restituisce il numero totale di utenti
func (r *AdminRepository) GetTotalUsersCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("errore nel conteggio utenti: %v", err)
	}
	return count, nil
}

// GetUserStats restituisce statistiche dettagliate di un utente
func (r *AdminRepository) GetUserStats(userID int64) (map[string]interface{}, error) {
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

	err := r.db.QueryRow(query, userID).Scan(
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
	err = r.db.QueryRow(
		"SELECT COUNT(*) FROM event_participants WHERE user_id = $1",
		userID,
	).Scan(&eventCount)
	if err == nil {
		stats["events_participated"] = eventCount
	}

	// Conta preferiti
	var favoritesCount int
	err = r.db.QueryRow(
		"SELECT COUNT(*) FROM user_favorites WHERE user_id = $1",
		userID,
	).Scan(&favoritesCount)
	if err == nil {
		stats["favorites_count"] = favoritesCount
	}

	// Conta amici
	var friendsCount int
	err = r.db.QueryRow(
		"SELECT COUNT(*) FROM friendships WHERE user1_id = $1 OR user2_id = $1",
		userID,
	).Scan(&friendsCount)
	if err == nil {
		stats["friends_count"] = friendsCount
	}

	return stats, nil
}