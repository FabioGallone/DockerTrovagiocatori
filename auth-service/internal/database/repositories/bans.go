package repositories

import (
	"database/sql"
	"fmt"

	"trovagiocatoriAuth/internal/models"
)

type BanRepository struct {
	db *sql.DB
}

func NewBanRepository(db *sql.DB) *BanRepository {
	return &BanRepository{db: db}
}

// BanUser banna un utente
func (r *BanRepository) BanUser(ban *models.BanUserRequest, adminID int64) (*models.UserBan, error) {
	tx, err := r.db.Begin() // Una transazione è un insieme di operazioni sul database che devono essere eseguite tutte insieme o nessuna.
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

	reason := ban.Reason
	if reason == "" {
		reason = "Ban amministrativo"
	}

	// Inserisci il nuovo ban
	var banID int64
	err = tx.QueryRow(`
		INSERT INTO user_bans (user_id, banned_by_admin_id, reason, notes)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		ban.UserID, adminID, reason, ban.Notes,
	).Scan(&banID)

	if err != nil {
		return nil, fmt.Errorf("errore nell'inserimento del ban: %v", err)
	}

	// Disattiva l'utente
	_, err = tx.Exec("UPDATE users SET is_active = FALSE WHERE id = $1", ban.UserID)
	if err != nil {
		return nil, fmt.Errorf("errore nella disattivazione dell'utente: %v", err)
	}

	// Aggiungi alla tabella ban_history
	_, err = tx.Exec(`
		INSERT INTO ban_history (user_id, admin_id, action, reason, ban_id)
		VALUES ($1, $2, 'banned', $3, $4)`,
		ban.UserID, adminID, reason, banID)
	if err != nil {
		fmt.Printf("Warning: errore inserimento cronologia: %v\n", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	// Ban appena creato con tutte le informazioni
	return r.GetBanByID(banID)
}

// UnbanUser rimuove il ban di un utente
func (r *BanRepository) UnbanUser(userID, adminID int64, reason string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if reason == "" {
		reason = "Ban rimosso dall'amministratore"
	}

	// Aggiorna tutti i ban attivi dell'utente
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
	}

	return tx.Commit()
}

// IsUserBanned controlla se un utente è attualmente bannato
func (r *BanRepository) IsUserBanned(userID int64) (bool, *models.UserBan, error) {
	ban, err := r.GetActiveBanByUserID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil, nil
		}
		return false, nil, err
	}

	return true, ban, nil
}

// GetActiveBanByUserID ottiene il ban attivo di un utente
func (r *BanRepository) GetActiveBanByUserID(userID int64) (*models.UserBan, error) {
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

	ban := &models.UserBan{}
	err := r.db.QueryRow(query, userID).Scan(
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
func (r *BanRepository) GetBanByID(banID int64) (*models.UserBan, error) {
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

	ban := &models.UserBan{}
	err := r.db.QueryRow(query, banID).Scan(
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
func (r *BanRepository) GetAllActiveBans() ([]models.UserBan, error) {
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

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bans []models.UserBan
	for rows.Next() {
		var ban models.UserBan
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
func (r *BanRepository) GetUserBanHistory(userID int64) ([]map[string]interface{}, error) {
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

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var id, userIDVal, adminID int64
		var action, reason, username string
		var createdAt string
		var banID sql.NullInt64
		var adminUsername sql.NullString

		err := rows.Scan(
			&id, &userIDVal, &adminID, &action, &reason, &createdAt,
			&banID, &username, &adminUsername,
		)
		if err != nil {
			return nil, err
		}

		adminName := "Sistema"
		if adminUsername.Valid {
			adminName = adminUsername.String
		}

		h := map[string]interface{}{
			"id":             id,
			"user_id":        userIDVal,
			"admin_id":       adminID,
			"action":         action,
			"reason":         reason,
			"created_at":     createdAt,
			"username":       username,
			"admin_username": adminName,
		}

		if banID.Valid {
			h["ban_id"] = banID.Int64
		}

		history = append(history, h)
	}

	return history, rows.Err()
}

// GetBanStats ottiene statistiche sui ban
func (r *BanRepository) GetBanStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Conta ban attivi
	var activeBans int
	err := r.db.QueryRow("SELECT COUNT(*) FROM user_bans WHERE is_active = TRUE").Scan(&activeBans)
	if err != nil {
		activeBans = 0
	}
	stats["active_bans"] = activeBans

	// Conta ban totali
	var totalBans int
	err = r.db.QueryRow("SELECT COUNT(*) FROM user_bans").Scan(&totalBans)
	if err != nil {
		totalBans = 0
	}
	stats["total_bans"] = totalBans

	// Conta ban rimossi oggi
	var unbannedToday int
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM user_bans 
		WHERE unbanned_at::date = CURRENT_DATE AND is_active = FALSE`).Scan(&unbannedToday)
	if err != nil {
		unbannedToday = 0
	}
	stats["unbanned_today"] = unbannedToday

	return stats, nil
}

