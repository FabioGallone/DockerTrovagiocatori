package repositories

import (
	"database/sql"
	"fmt"


	"trovagiocatoriAuth/internal/models"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// AddFavorite aggiunge un post ai preferiti dell'utente
func (r *EventRepository) AddFavorite(userID int64, postID int) error {
	_, err := r.db.Exec(`
		INSERT INTO user_favorites (user_id, post_id) 
		VALUES ($1, $2) 
		ON CONFLICT (user_id, post_id) DO NOTHING`,
		userID, postID)
	return err
}

// RemoveFavorite rimuove un post dai preferiti dell'utente
func (r *EventRepository) RemoveFavorite(userID int64, postID int) error {
	_, err := r.db.Exec(`
		DELETE FROM user_favorites 
		WHERE user_id = $1 AND post_id = $2`,
		userID, postID)
	return err
}

// IsFavorite controlla se un post è nei preferiti dell'utente
func (r *EventRepository) IsFavorite(userID int64, postID int) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM user_favorites 
		WHERE user_id = $1 AND post_id = $2`,
		userID, postID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetUserFavorites restituisce tutti i post ID preferiti di un utente
func (r *EventRepository) GetUserFavorites(userID int64) ([]int, error) {
	rows, err := r.db.Query(`
		SELECT post_id FROM user_favorites 
		WHERE user_id = $1 
		ORDER BY created_at DESC`,
		userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favorites []int
	for rows.Next() {
		var postID int
		if err := rows.Scan(&postID); err != nil {
			return nil, err
		}
		favorites = append(favorites, postID)
	}

	return favorites, rows.Err()
}

// JoinEvent - Iscrive un utente a un evento
func (r *EventRepository) JoinEvent(userID int64, postID int) error {
	// Controlla se l'utente è già iscritto
	isAlreadyParticipant, err := r.IsEventParticipant(userID, postID)
	if err != nil {
		return err
	}
	
	if isAlreadyParticipant {
		return fmt.Errorf("utente già iscritto a questo evento")
	}

	_, err = r.db.Exec(`
		INSERT INTO event_participants (user_id, post_id, status) 
		VALUES ($1, $2, 'confirmed')`,
		userID, postID)
	return err
}

// LeaveEvent - Disiscrive un utente da un evento
func (r *EventRepository) LeaveEvent(userID int64, postID int) error {
	_, err := r.db.Exec(`
		DELETE FROM event_participants 
		WHERE user_id = $1 AND post_id = $2`,
		userID, postID)
	return err
}

// IsEventParticipant - Controlla se un utente è iscritto a un evento
func (r *EventRepository) IsEventParticipant(userID int64, postID int) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM event_participants 
		WHERE user_id = $1 AND post_id = $2 AND status = 'confirmed'`,
		userID, postID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetEventParticipants - Ottiene tutti i partecipanti di un evento con le loro informazioni
func (r *EventRepository) GetEventParticipants(postID int) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
		SELECT 
			u.id, u.username, u.nome, u.cognome, u.email, u.profile_picture,
			ep.registered_at
		FROM event_participants ep
		JOIN users u ON ep.user_id = u.id
		WHERE ep.post_id = $1 AND ep.status = 'confirmed'
		ORDER BY ep.registered_at ASC`,
		postID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []map[string]interface{}
	for rows.Next() {
		var userID int64
		var username, nome, cognome, email, profilePic, registeredAt string
		
		err := rows.Scan(&userID, &username, &nome, &cognome, &email, &profilePic, &registeredAt)
		if err != nil {
			return nil, err
		}

		participant := map[string]interface{}{
			"user_id":         userID,
			"username":        username,
			"nome":            nome,
			"cognome":         cognome,
			"email":           email,
			"profile_picture": profilePic,
			"registered_at":   registeredAt,
		}
		participants = append(participants, participant)
	}

	return participants, rows.Err()
}

// GetEventParticipantCount - Ottiene il numero di partecipanti iscritti a un evento
func (r *EventRepository) GetEventParticipantCount(postID int) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM event_participants 
		WHERE post_id = $1 AND status = 'confirmed'`,
		postID).Scan(&count)

	return count, err
}

// GetUserParticipations - Ottiene tutti gli eventi a cui un utente è iscritto
func (r *EventRepository) GetUserParticipations(userID int64) ([]int, error) {
	rows, err := r.db.Query(`
		SELECT post_id FROM event_participants 
		WHERE user_id = $1 AND status = 'confirmed'
		ORDER BY registered_at DESC`,
		userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participations []int
	for rows.Next() {
		var postID int
		if err := rows.Scan(&postID); err != nil {
			return nil, err
		}
		participations = append(participations, postID)
	}

	return participations, rows.Err()
}

// SendEventInvite - Invia un invito per un evento
func (r *EventRepository) SendEventInvite(senderID, receiverID int64, postID int, message string) error {
	_, err := r.db.Exec(`
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
func (r *EventRepository) CheckPendingEventInvite(receiverID int64, postID int) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM event_invites 
		WHERE receiver_id = $1 AND post_id = $2 AND status = 'pending'`,
		receiverID, postID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// AcceptEventInvite - Accetta un invito per un evento e iscrive automaticamente l'utente
func (r *EventRepository) AcceptEventInvite(inviteID, receiverID int64) error {
	tx, err := r.db.Begin()
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
func (r *EventRepository) RejectEventInvite(inviteID, receiverID int64) error {
	// Verifica che l'utente sia il destinatario dell'invito
	var actualReceiverID int64
	err := r.db.QueryRow(`
		SELECT receiver_id FROM event_invites 
		WHERE id = $1 AND status = 'pending'`, inviteID).Scan(&actualReceiverID)

	if err != nil {
		return fmt.Errorf("invito non trovato o non più pendente: %v", err)
	}

	if actualReceiverID != receiverID {
		return fmt.Errorf("non autorizzato a rifiutare questo invito")
	}

	// Aggiorna lo status dell'invito
	_, err = r.db.Exec(`
		UPDATE event_invites 
		SET status = 'rejected', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1`, inviteID)

	return err
}

// GetEventInvites - Ottiene gli inviti ricevuti da un utente
func (r *EventRepository) GetEventInvites(userID int64) ([]models.EventInviteInfo, error) {
	rows, err := r.db.Query(`
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

	var invites []models.EventInviteInfo
	for rows.Next() {
		var invite models.EventInviteInfo
		var createdAt string

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

		invite.CreatedAt = createdAt
		invites = append(invites, invite)
	}

	return invites, rows.Err()
}

// GetAvailableFriendsForInvite ottiene la lista degli amici che possono essere invitati a un evento
func (r *EventRepository) GetAvailableFriendsForInvite(userID int64, postID int) ([]models.FriendInfo, error) {
	query := `
		SELECT 
			CASE 
				WHEN f.user1_id = $1 THEN u2.id
				ELSE u1.id
			END as friend_id,
			CASE 
				WHEN f.user1_id = $1 THEN u2.username
				ELSE u1.username
			END as username,
			CASE 
				WHEN f.user1_id = $1 THEN u2.nome
				ELSE u1.nome
			END as nome,
			CASE 
				WHEN f.user1_id = $1 THEN u2.cognome
				ELSE u1.cognome
			END as cognome,
			CASE 
				WHEN f.user1_id = $1 THEN u2.email
				ELSE u1.email
			END as email,
			CASE 
				WHEN f.user1_id = $1 THEN u2.profile_picture
				ELSE u1.profile_picture
			END as profile_picture,
			f.created_at
		FROM friendships f
		JOIN users u1 ON f.user1_id = u1.id
		JOIN users u2 ON f.user2_id = u2.id
		WHERE (f.user1_id = $1 OR f.user2_id = $1)
		AND CASE 
			WHEN f.user1_id = $1 THEN u2.id
			ELSE u1.id
		END NOT IN (
			-- Escludi utenti già invitati
			SELECT ei.receiver_id 
			FROM event_invites ei 
			WHERE ei.post_id = $2 AND ei.status = 'pending'
		)
		AND CASE 
			WHEN f.user1_id = $1 THEN u2.id
			ELSE u1.id
		END NOT IN (
			-- Escludi utenti già partecipanti
			SELECT ep.user_id 
			FROM event_participants ep 
			WHERE ep.post_id = $2 AND ep.status = 'confirmed'
		)
		ORDER BY f.created_at DESC`

	rows, err := r.db.Query(query, userID, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []models.FriendInfo
	for rows.Next() {
		var friend models.FriendInfo
		var createdAt string

		err := rows.Scan(
			&friend.UserID,
			&friend.Username,
			&friend.Nome,
			&friend.Cognome,
			&friend.Email,
			&friend.ProfilePic,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		friend.FriendsSince = createdAt
		friends = append(friends, friend)
	}

	return friends, rows.Err()
}

// GetEventInvitePostID ottiene l'ID del post associato a un invito evento
func (r *EventRepository) GetEventInvitePostID(inviteID int64) (int64, error) {
	var postID int64
	query := `SELECT post_id FROM event_invites WHERE id = $1`

	err := r.db.QueryRow(query, inviteID).Scan(&postID)
	if err != nil {
		return 0, err
	}
	return postID, nil
}

// GetPostTitleByID ottiene il titolo di un post tramite il suo ID
func (r *EventRepository) GetPostTitleByID(postID int) (string, error) {
	// Nota: Questo metodo richiede accesso al database Python
	// Per ora restituiamo un placeholder
	return fmt.Sprintf("Evento #%d", postID), nil
}

// GetInvitedUserEmailsForPost ottiene le email degli utenti già invitati a un evento
func (r *EventRepository) GetInvitedUserEmailsForPost(postID int) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT u.email 
		FROM event_invites ei
		JOIN users u ON ei.receiver_id = u.id
		WHERE ei.post_id = $1 AND ei.status = 'pending'`,
		postID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, rows.Err()
}

// GetParticipantEmailsForPost ottiene le email degli utenti già partecipanti a un evento
func (r *EventRepository) GetParticipantEmailsForPost(postID int) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT u.email 
		FROM event_participants ep
		JOIN users u ON ep.user_id = u.id
		WHERE ep.post_id = $1 AND ep.status = 'confirmed'`,
		postID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, rows.Err()
}