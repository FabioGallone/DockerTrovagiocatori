// auth-service/internal/db/event_invites.go
package db

import (
	"fmt"
	"time"
)

// Aggiungi questa funzione al file friends.go o crea un nuovo file event_invites.go

// CreateEventInvitesTableIfNotExists - Crea la tabella per gli inviti eventi se non esiste
func (db *Database) CreateEventInvitesTableIfNotExists() error {
	_, err := db.Conn.Exec(`
	CREATE TABLE IF NOT EXISTS event_invites (
		id SERIAL PRIMARY KEY,
		sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		receiver_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		post_id INTEGER NOT NULL,
		message TEXT,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(receiver_id, post_id),
		CHECK(sender_id != receiver_id),
		CHECK(status IN ('pending', 'accepted', 'rejected'))
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella event_invites: %v", err)
	}

	// Indici per migliorare le performance
	_, err = db.Conn.Exec(`
	CREATE INDEX IF NOT EXISTS idx_event_invites_sender ON event_invites(sender_id);
	CREATE INDEX IF NOT EXISTS idx_event_invites_receiver ON event_invites(receiver_id);
	CREATE INDEX IF NOT EXISTS idx_event_invites_post ON event_invites(post_id);
	CREATE INDEX IF NOT EXISTS idx_event_invites_status ON event_invites(status);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione degli indici event_invites: %v", err)
	}

	fmt.Println("✔ Tabella event_invites creata/verificata con successo!")
	return nil
}

// SendEventInvite - Invia un invito per un evento
func (db *Database) SendEventInvite(senderID, receiverID int64, postID int, message string) error {
	_, err := db.Conn.Exec(`
		INSERT INTO event_invites (sender_id, receiver_id, post_id, message, status) 
		VALUES ($1, $2, $3, $4, 'pending')
		ON CONFLICT (receiver_id, post_id) 
		DO UPDATE SET 
			status = 'pending', 
			message = $4,
			updated_at = CURRENT_TIMESTAMP`,
		senderID, receiverID, postID, message)
	return err
}

// CheckPendingEventInvite - Controlla se esiste un invito pendente per un evento
func (db *Database) CheckPendingEventInvite(receiverID int64, postID int) (bool, error) {
	var count int
	err := db.Conn.QueryRow(`
		SELECT COUNT(*) FROM event_invites 
		WHERE receiver_id = $1 AND post_id = $2 AND status = 'pending'`,
		receiverID, postID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// AcceptEventInvite - Accetta un invito per un evento e iscrive automaticamente l'utente
func (db *Database) AcceptEventInvite(inviteID, receiverID int64) error {
	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ottieni i dettagli dell'invito
	var senderID, actualReceiverID int64
	var postID int
	var status string
	err = tx.QueryRow(`
		SELECT sender_id, receiver_id, post_id, status 
		FROM event_invites 
		WHERE id = $1`, inviteID).Scan(&senderID, &actualReceiverID, &postID, &status)

	if err != nil {
		return fmt.Errorf("invito non trovato: %v", err)
	}

	// Verifica che l'utente sia il destinatario dell'invito
	if actualReceiverID != receiverID {
		return fmt.Errorf("non autorizzato ad accettare questo invito")
	}

	// Verifica che l'invito sia ancora pendente
	if status != "pending" {
		return fmt.Errorf("l'invito non è più pendente")
	}

	// Aggiorna lo status dell'invito
	_, err = tx.Exec(`
		UPDATE event_invites 
		SET status = 'accepted', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1`, inviteID)
	if err != nil {
		return err
	}

	// Iscrive automaticamente l'utente all'evento (se non è già iscritto)
	_, err = tx.Exec(`
		INSERT INTO event_participants (user_id, post_id, status) 
		VALUES ($1, $2, 'confirmed')
		ON CONFLICT (user_id, post_id) DO NOTHING`,
		receiverID, postID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RejectEventInvite - Rifiuta un invito per un evento
func (db *Database) RejectEventInvite(inviteID, receiverID int64) error {
	// Verifica che l'utente sia il destinatario dell'invito
	var actualReceiverID int64
	err := db.Conn.QueryRow(`
		SELECT receiver_id FROM event_invites 
		WHERE id = $1 AND status = 'pending'`, inviteID).Scan(&actualReceiverID)

	if err != nil {
		return fmt.Errorf("invito non trovato o non più pendente: %v", err)
	}

	if actualReceiverID != receiverID {
		return fmt.Errorf("non autorizzato a rifiutare questo invito")
	}

	// Aggiorna lo status dell'invito
	_, err = db.Conn.Exec(`
		UPDATE event_invites 
		SET status = 'rejected', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1`, inviteID)

	return err
}

// GetEventInvites - Ottiene gli inviti ricevuti da un utente
func (db *Database) GetEventInvites(userID int64) ([]EventInviteInfo, error) {
	rows, err := db.Conn.Query(`
		SELECT 
			ei.id,
			ei.post_id,
			ei.message,
			ei.created_at,
			ei.status,
			u.username as sender_username,
			u.nome as sender_nome,
			u.cognome as sender_cognome,
			u.email as sender_email,
			u.profile_picture as sender_profile_picture
		FROM event_invites ei
		JOIN users u ON ei.sender_id = u.id
		WHERE ei.receiver_id = $1 AND ei.status = 'pending'
		ORDER BY ei.created_at DESC`,
		userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []EventInviteInfo
	for rows.Next() {
		var invite EventInviteInfo
		var createdAt time.Time

		err := rows.Scan(
			&invite.InviteID,
			&invite.PostID,
			&invite.Message,
			&createdAt,
			&invite.Status,
			&invite.SenderUsername,
			&invite.SenderNome,
			&invite.SenderCognome,
			&invite.SenderEmail,
			&invite.SenderProfilePicture,
		)
		if err != nil {
			return nil, err
		}

		invite.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		invites = append(invites, invite)
	}

	return invites, rows.Err()
}

// GetUserUnreadEventInvitesCount ottiene il numero di inviti evento non letti
func (db *Database) GetUserUnreadEventInvitesCount(userID int64) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM event_invites 
		WHERE receiver_id = $1 AND status = 'pending'`

	err := db.Conn.QueryRow(query, userID).Scan(&count)
	return count, err
}

// Struttura per le informazioni dell'invito evento
type EventInviteInfo struct {
	InviteID             int64  `json:"invite_id"`
	PostID               int    `json:"post_id"`
	Message              string `json:"message"`
	CreatedAt            string `json:"created_at"`
	Status               string `json:"status"`
	SenderUsername       string `json:"sender_username"`
	SenderNome           string `json:"sender_nome"`
	SenderCognome        string `json:"sender_cognome"`
	SenderEmail          string `json:"sender_email"`
	SenderProfilePicture string `json:"sender_profile_picture"`
}

// GetEventInvitePostID ottiene l'ID del post associato a un invito evento
func (db *Database) GetEventInvitePostID(inviteID int64) (int64, error) {
	var postID int64
	query := `SELECT post_id FROM event_invites WHERE id = $1`

	err := db.Conn.QueryRow(query, inviteID).Scan(&postID)
	if err != nil {
		return 0, err
	}
	return postID, nil
}

// GetPostTitleByID ottiene il titolo di un post tramite il suo ID
func (db *Database) GetPostTitleByID(postID int) (string, error) {
	// Nota: Questo metodo richiede accesso al database Python
	// Per ora restituiamo un placeholder, ma dovresti implementare
	// una tabella locale o fare una chiamata al servizio Python
	return fmt.Sprintf("Evento #%d", postID), nil
}

// GetEventInviteDetails ottiene i dettagli completi di un invito evento
func (db *Database) GetEventInviteDetails(inviteID int64) (*EventInviteDetails, error) {
	query := `
		SELECT 
			ei.id, ei.sender_id, ei.receiver_id, ei.post_id, ei.message, ei.status, ei.created_at,
			u_sender.username as sender_username, u_sender.nome as sender_nome, u_sender.cognome as sender_cognome,
			u_receiver.username as receiver_username, u_receiver.nome as receiver_nome, u_receiver.cognome as receiver_cognome
		FROM event_invites ei
		JOIN users u_sender ON ei.sender_id = u_sender.id
		JOIN users u_receiver ON ei.receiver_id = u_receiver.id
		WHERE ei.id = $1`

	var details EventInviteDetails
	err := db.Conn.QueryRow(query, inviteID).Scan(
		&details.ID,
		&details.SenderID,
		&details.ReceiverID,
		&details.PostID,
		&details.Message,
		&details.Status,
		&details.CreatedAt,
		&details.SenderUsername,
		&details.SenderNome,
		&details.SenderCognome,
		&details.ReceiverUsername,
		&details.ReceiverNome,
		&details.ReceiverCognome,
	)

	if err != nil {
		return nil, err
	}

	return &details, nil
}


type EventInviteDetails struct {
	ID               int64     `json:"id"`
	SenderID         int64     `json:"sender_id"`
	ReceiverID       int64     `json:"receiver_id"`
	PostID           int       `json:"post_id"`
	Message          string    `json:"message"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	SenderUsername   string    `json:"sender_username"`
	SenderNome       string    `json:"sender_nome"`
	SenderCognome    string    `json:"sender_cognome"`
	ReceiverUsername string    `json:"receiver_username"`
	ReceiverNome     string    `json:"receiver_nome"`
	ReceiverCognome  string    `json:"receiver_cognome"`
}
