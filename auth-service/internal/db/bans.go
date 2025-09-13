// auth-service/internal/db/bans.go
package db

import (
	"database/sql"
	"fmt"
	"time"
)

// UserBan rappresenta un ban di un utente
type UserBan struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	BannedByAdminID int64      `json:"banned_by_admin_id"`
	Reason          string     `json:"reason"`
	BannedAt        time.Time  `json:"banned_at"`
	UnbannedAt      *time.Time `json:"unbanned_at,omitempty"`
	UnbannedByID    *int64     `json:"unbanned_by_admin_id,omitempty"`
	IsActive        bool       `json:"is_active"`
	Notes           string     `json:"notes"`

	// Informazioni aggiuntive per la visualizzazione
	Username      string `json:"username"`
	Email         string `json:"email"`
	AdminUsername string `json:"admin_username"`
}

// BanHistory rappresenta una voce nella cronologia dei ban
type BanHistory struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	AdminID   int64     `json:"admin_id"`
	Action    string    `json:"action"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
	BanID     *int64    `json:"ban_id,omitempty"`

	// Informazioni aggiuntive
	Username      string `json:"username"`
	AdminUsername string `json:"admin_username"`
}

// BanUserRequest rappresenta una richiesta di ban
type BanUserRequest struct {
	UserID int64  `json:"user_id"`
	Reason string `json:"reason"`
	Notes  string `json:"notes"`
}

// CreateBanTablesIfNotExists crea le tabelle per i ban se non esistono
func (db *Database) CreateBanTablesIfNotExists() error {
	fmt.Println("🔧 Inizializzazione sistema ban...")

	// 1. Crea tabella user_bans SEMPLIFICATA
	_, err := db.Conn.Exec(`
		CREATE TABLE IF NOT EXISTS user_bans (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			banned_by_admin_id INTEGER NOT NULL REFERENCES users(id) ON DELETE SET NULL,
			reason TEXT DEFAULT 'Ban amministrativo',
			banned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			unbanned_at TIMESTAMP NULL,
			unbanned_by_admin_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			is_active BOOLEAN DEFAULT TRUE,
			notes TEXT
		)`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella user_bans: %v", err)
	}

	// 2. Crea tabella ban_history
	_, err = db.Conn.Exec(`
		CREATE TABLE IF NOT EXISTS ban_history (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			admin_id INTEGER NOT NULL REFERENCES users(id) ON DELETE SET NULL,
			action VARCHAR(20) NOT NULL,
			reason TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			ban_id INTEGER REFERENCES user_bans(id) ON DELETE SET NULL
		)`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella ban_history: %v", err)
	}

	// 3. Crea indici per performance
	db.Conn.Exec(`CREATE INDEX IF NOT EXISTS idx_user_bans_user_id ON user_bans(user_id)`)
	db.Conn.Exec(`CREATE INDEX IF NOT EXISTS idx_user_bans_active ON user_bans(is_active)`)
	db.Conn.Exec(`CREATE INDEX IF NOT EXISTS idx_ban_history_user_id ON ban_history(user_id)`)
	db.Conn.Exec(`CREATE INDEX IF NOT EXISTS idx_ban_history_created_at ON ban_history(created_at)`)

	fmt.Println("✔ Tabelle ban create/verificate con successo!")
	return nil
}

// BanUser banna un utente
func (db *Database) BanUser(ban *BanUserRequest, adminID int64, ipAddress, userAgent string) (*UserBan, error) {
	tx, err := db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Controlla se l'utente esiste
	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", ban.UserID).Scan(&exists)
	if err != nil || !exists {
		return nil, fmt.Errorf("utente non trovato")
	}

	// Controlla se c'è già un ban attivo per questo utente
	var activeBanCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM user_bans WHERE user_id = $1 AND is_active = TRUE", ban.UserID).Scan(&activeBanCount)
	if err != nil {
		return nil, err
	}

	if activeBanCount > 0 {
		return nil, fmt.Errorf("utente già bannato")
	}

	// Inserisci il nuovo ban (SEMPLIFICATO)
	var banID int64
	err = tx.QueryRow(`
		INSERT INTO user_bans (user_id, banned_by_admin_id, reason, notes)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		ban.UserID, adminID, ban.Reason, ban.Notes,
	).Scan(&banID)

	if err != nil {
		return nil, fmt.Errorf("errore nell'inserimento del ban: %v", err)
	}

	// Disattiva l'utente
	_, err = tx.Exec("UPDATE users SET is_active = FALSE WHERE id = $1", ban.UserID)
	if err != nil {
		return nil, fmt.Errorf("errore nella disattivazione dell'utente: %v", err)
	}

	// Aggiungi alla cronologia
	_, err = tx.Exec(`
		INSERT INTO ban_history (user_id, admin_id, action, reason, ban_id)
		VALUES ($1, $2, 'banned', $3, $4)`,
		ban.UserID, adminID, ban.Reason, banID)
	if err != nil {
		fmt.Printf("Warning: errore inserimento cronologia: %v\n", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	// Ottieni il ban appena creato con tutte le informazioni
	return db.GetBanByID(banID)
}

// UnbanUser rimuove il ban di un utente
func (db *Database) UnbanUser(userID, adminID int64, reason string) error {
	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Assicurati che reason non sia vuoto
	if reason == "" {
		reason = "Ban rimosso dall'amministratore"
	}

	// Aggiorna tutti i ban attivi dell'utente - FIX: cast esplicito del tipo
	result, err := tx.Exec(`
		UPDATE user_bans 
		SET is_active = FALSE, 
		    unbanned_at = CURRENT_TIMESTAMP,
		    unbanned_by_admin_id = $1,
		    notes = CONCAT(COALESCE(notes, ''), ' - ', $2::text)
		WHERE user_id = $3 AND is_active = TRUE`,
		adminID, reason, userID)

	if err != nil {
		return fmt.Errorf("errore nell'aggiornamento del ban: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("nessun ban attivo trovato per questo utente")
	}

	// Riattiva l'utente
	_, err = tx.Exec("UPDATE users SET is_active = TRUE WHERE id = $1", userID)
	if err != nil {
		return fmt.Errorf("errore nella riattivazione dell'utente: %v", err)
	}

	// Aggiungi alla cronologia
	_, err = tx.Exec(`
		INSERT INTO ban_history (user_id, admin_id, action, reason)
		VALUES ($1, $2, 'unbanned', $3)`,
		userID, adminID, reason)
	if err != nil {
		fmt.Printf("Warning: errore inserimento cronologia unban: %v\n", err)
		// Non interrompiamo il flusso
	}

	return tx.Commit()
}

// IsUserBanned controlla se un utente è attualmente bannato
func (db *Database) IsUserBanned(userID int64) (bool, *UserBan, error) {
	ban, err := db.GetActiveBanByUserID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil, nil
		}
		return false, nil, err
	}

	// Per i ban permanenti, è sempre attivo
	return true, ban, nil
}

// GetActiveBanByUserID ottiene il ban attivo di un utente
func (db *Database) GetActiveBanByUserID(userID int64) (*UserBan, error) {
	query := `
		SELECT 
			ub.id, ub.user_id, ub.banned_by_admin_id, ub.reason, ub.banned_at,
			ub.unbanned_at, ub.unbanned_by_admin_id, ub.is_active, ub.notes,
			u.username, u.email,
			admin.username as admin_username
		FROM user_bans ub
		JOIN users u ON ub.user_id = u.id
		JOIN users admin ON ub.banned_by_admin_id = admin.id
		WHERE ub.user_id = $1 AND ub.is_active = TRUE
		ORDER BY ub.banned_at DESC
		LIMIT 1`

	ban := &UserBan{}
	err := db.Conn.QueryRow(query, userID).Scan(
		&ban.ID, &ban.UserID, &ban.BannedByAdminID, &ban.Reason, &ban.BannedAt,
		&ban.UnbannedAt, &ban.UnbannedByID, &ban.IsActive, &ban.Notes,
		&ban.Username, &ban.Email, &ban.AdminUsername,
	)

	if err != nil {
		return nil, err
	}

	return ban, nil
}

// GetBanByID ottiene un ban per ID
func (db *Database) GetBanByID(banID int64) (*UserBan, error) {
	query := `
		SELECT 
			ub.id, ub.user_id, ub.banned_by_admin_id, ub.reason, ub.banned_at,
			ub.unbanned_at, ub.unbanned_by_admin_id, ub.is_active, ub.notes,
			u.username, u.email,
			admin.username as admin_username
		FROM user_bans ub
		JOIN users u ON ub.user_id = u.id
		JOIN users admin ON ub.banned_by_admin_id = admin.id
		WHERE ub.id = $1`

	ban := &UserBan{}
	err := db.Conn.QueryRow(query, banID).Scan(
		&ban.ID, &ban.UserID, &ban.BannedByAdminID, &ban.Reason, &ban.BannedAt,
		&ban.UnbannedAt, &ban.UnbannedByID, &ban.IsActive, &ban.Notes,
		&ban.Username, &ban.Email, &ban.AdminUsername,
	)

	if err != nil {
		return nil, err
	}

	return ban, nil
}

// GetAllActiveBans ottiene tutti i ban attivi
func (db *Database) GetAllActiveBans() ([]UserBan, error) {
	query := `
		SELECT 
			ub.id, ub.user_id, ub.banned_by_admin_id, ub.reason, ub.banned_at,
			ub.unbanned_at, ub.unbanned_by_admin_id, ub.is_active, ub.notes,
			u.username, u.email,
			admin.username as admin_username
		FROM user_bans ub
		JOIN users u ON ub.user_id = u.id
		JOIN users admin ON ub.banned_by_admin_id = admin.id
		WHERE ub.is_active = TRUE
		ORDER BY ub.banned_at DESC`

	rows, err := db.Conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bans []UserBan
	for rows.Next() {
		var ban UserBan
		err := rows.Scan(
			&ban.ID, &ban.UserID, &ban.BannedByAdminID, &ban.Reason, &ban.BannedAt,
			&ban.UnbannedAt, &ban.UnbannedByID, &ban.IsActive, &ban.Notes,
			&ban.Username, &ban.Email, &ban.AdminUsername,
		)
		if err != nil {
			return nil, err
		}
		bans = append(bans, ban)
	}

	return bans, rows.Err()
}

// GetUserBanHistory ottiene la cronologia dei ban di un utente
func (db *Database) GetUserBanHistory(userID int64) ([]BanHistory, error) {
	query := `
		SELECT 
			bh.id, bh.user_id, bh.admin_id, bh.action, bh.reason, bh.created_at,
			bh.ban_id,
			u.username, admin.username as admin_username
		FROM ban_history bh
		JOIN users u ON bh.user_id = u.id
		LEFT JOIN users admin ON bh.admin_id = admin.id
		WHERE bh.user_id = $1
		ORDER BY bh.created_at DESC`

	rows, err := db.Conn.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []BanHistory
	for rows.Next() {
		var h BanHistory
		var adminUsername sql.NullString

		err := rows.Scan(
			&h.ID, &h.UserID, &h.AdminID, &h.Action, &h.Reason, &h.CreatedAt,
			&h.BanID,
			&h.Username, &adminUsername,
		)
		if err != nil {
			return nil, err
		}

		if adminUsername.Valid {
			h.AdminUsername = adminUsername.String
		} else {
			h.AdminUsername = "Sistema"
		}

		history = append(history, h)
	}

	return history, rows.Err()
}

// GetBanStats ottiene statistiche sui ban
func (db *Database) GetBanStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Conta ban attivi
	var activeBans int
	err := db.Conn.QueryRow("SELECT COUNT(*) FROM user_bans WHERE is_active = TRUE").Scan(&activeBans)
	if err != nil {
		activeBans = 0
	}
	stats["active_bans"] = activeBans

	// Conta ban totali
	var totalBans int
	err = db.Conn.QueryRow("SELECT COUNT(*) FROM user_bans").Scan(&totalBans)
	if err != nil {
		totalBans = 0
	}
	stats["total_bans"] = totalBans

	// Conta ban rimossi oggi
	var unbannedToday int
	err = db.Conn.QueryRow(`
		SELECT COUNT(*) FROM user_bans 
		WHERE unbanned_at::date = CURRENT_DATE AND is_active = FALSE`).Scan(&unbannedToday)
	if err != nil {
		unbannedToday = 0
	}
	stats["unbanned_today"] = unbannedToday

	return stats, nil
}

// CleanupExpiredBans pulisce automaticamente i ban scaduti
func (db *Database) CleanupExpiredBans() error {
	_, err := db.Conn.Exec(`
		UPDATE user_bans 
		SET is_active = FALSE, 
		    unbanned_at = CURRENT_TIMESTAMP,
		    notes = CONCAT(COALESCE(notes, ''), ' - Auto-unbanned: expired')
		WHERE is_active = TRUE 
		AND expires_at IS NOT NULL 
		AND expires_at < CURRENT_TIMESTAMP`)

	return err
}
