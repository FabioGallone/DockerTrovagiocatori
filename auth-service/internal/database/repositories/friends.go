package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"trovagiocatoriAuth/internal/models"
)

type FriendRepository struct {
	db *sql.DB
}

func NewFriendRepository(db *sql.DB) *FriendRepository {
	return &FriendRepository{db: db}
}

// SendFriendRequest - Invia una richiesta di amicizia
func (r *FriendRepository) SendFriendRequest(senderID, receiverID int64) error {
	_, err := r.db.Exec(`
		INSERT INTO friend_requests (sender_id, receiver_id, status) 
		VALUES ($1, $2, 'pending')
		ON CONFLICT (sender_id, receiver_id) 
		DO UPDATE SET status = 'pending', updated_at = CURRENT_TIMESTAMP`,
		senderID, receiverID)
	return err
}

// CheckPendingFriendRequest - Controlla se esiste una richiesta di amicizia pendente
func (r *FriendRepository) CheckPendingFriendRequest(userID1, userID2 int64) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM friend_requests 
		WHERE ((sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1))
		AND status = 'pending'`,
		userID1, userID2).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CheckFriendship - Controlla se due utenti sono amici
func (r *FriendRepository) CheckFriendship(userID1, userID2 int64) (bool, error) {
	// user1_id < user2_id per la query
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM friendships 
		WHERE user1_id = $1 AND user2_id = $2`,
		userID1, userID2).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// AcceptFriendRequest - Accetta una richiesta di amicizia
func (r *FriendRepository) AcceptFriendRequest(requestID, receiverID int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ottieni i dettagli della richiesta
	var senderID, actualReceiverID int64
	var status string
	err = tx.QueryRow(`
		SELECT sender_id, receiver_id, status 
		FROM friend_requests 
		WHERE id = $1`, requestID).Scan(&senderID, &actualReceiverID, &status)

	if err != nil {
		return fmt.Errorf("richiesta non trovata: %v", err)
	}

	// Verifica che l'utente sia il destinatario della richiesta
	if actualReceiverID != receiverID {
		return fmt.Errorf("non autorizzato ad accettare questa richiesta")
	}

	// Verifica che la richiesta sia ancora pendente
	if status != "pending" {
		return fmt.Errorf("la richiesta non è più pendente")
	}

	// Aggiorna lo status della richiesta
	_, err = tx.Exec(`
		UPDATE friend_requests 
		SET status = 'accepted', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1`, requestID)
	if err != nil {
		return err
	}

	// Crea l'amicizia (user1_id < user2_id)
	user1ID, user2ID := senderID, actualReceiverID
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	_, err = tx.Exec(`
		INSERT INTO friendships (user1_id, user2_id) 
		VALUES ($1, $2)
		ON CONFLICT (user1_id, user2_id) DO NOTHING`,
		user1ID, user2ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RejectFriendRequest - Rifiuta una richiesta di amicizia
func (r *FriendRepository) RejectFriendRequest(requestID, receiverID int64) error {
	// Verifica che l'utente sia il destinatario della richiesta
	var actualReceiverID int64
	err := r.db.QueryRow(`
		SELECT receiver_id FROM friend_requests 
		WHERE id = $1 AND status = 'pending'`, requestID).Scan(&actualReceiverID)

	if err != nil {
		return fmt.Errorf("richiesta non trovata o non più pendente: %v", err)
	}

	if actualReceiverID != receiverID {
		return fmt.Errorf("non autorizzato a rifiutare questa richiesta")
	}

	// Aggiorna lo status della richiesta
	_, err = r.db.Exec(`
		UPDATE friend_requests 
		SET status = 'rejected', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1`, requestID)

	return err
}

// RemoveFriendship - Rimuove un'amicizia
func (r *FriendRepository) RemoveFriendship(userID1, userID2 int64) error {
	// user1_id < user2_id per la query
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	_, err := r.db.Exec(`
		DELETE FROM friendships 
		WHERE user1_id = $1 AND user2_id = $2`,
		userID1, userID2)

	return err
}

// GetFriendsList - Ottiene la lista degli amici di un utente
func (r *FriendRepository) GetFriendsList(userID int64) ([]models.FriendInfo, error) {
	rows, err := r.db.Query(`
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
		WHERE f.user1_id = $1 OR f.user2_id = $1
		ORDER BY f.created_at DESC`,
		userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []models.FriendInfo
	for rows.Next() {
		var friend models.FriendInfo
		var createdAt time.Time

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

		friend.FriendsSince = createdAt.Format("2006-01-02 15:04:05")
		friends = append(friends, friend)
	}

	return friends, rows.Err()
}

// GetFriendRequests - Ottiene le richieste di amicizia ricevute
func (r *FriendRepository) GetFriendRequests(userID int64) ([]models.FriendRequestInfo, error) {
	rows, err := r.db.Query(`
		SELECT 
			fr.id,
			u.id,
			u.username,
			u.nome,
			u.cognome,
			u.email,
			u.profile_picture,
			fr.created_at,
			fr.status
		FROM friend_requests fr
		JOIN users u ON fr.sender_id = u.id
		WHERE fr.receiver_id = $1 AND fr.status = 'pending'
		ORDER BY fr.created_at DESC`,
		userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []models.FriendRequestInfo
	for rows.Next() {
		var request models.FriendRequestInfo
		var createdAt time.Time

		err := rows.Scan(
			&request.RequestID,
			&request.UserID,
			&request.Username,
			&request.Nome,
			&request.Cognome,
			&request.Email,
			&request.ProfilePic,
			&createdAt,
			&request.Status,
		)
		if err != nil {
			return nil, err
		}

		request.RequestDate = createdAt.Format("2006-01-02 15:04:05")
		requests = append(requests, request)
	}

	return requests, rows.Err()
}

// GetSentFriendRequests - Ottiene le richieste di amicizia inviate
func (r *FriendRepository) GetSentFriendRequests(userID int64) ([]models.FriendRequestInfo, error) {
	rows, err := r.db.Query(`
		SELECT 
			fr.id,
			u.id,
			u.username,
			u.nome,
			u.cognome,
			u.email,
			u.profile_picture,
			fr.created_at,
			fr.status
		FROM friend_requests fr
		JOIN users u ON fr.receiver_id = u.id
		WHERE fr.sender_id = $1 AND fr.status = 'pending'
		ORDER BY fr.created_at DESC`,
		userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []models.FriendRequestInfo
	for rows.Next() {
		var request models.FriendRequestInfo
		var createdAt time.Time

		err := rows.Scan(
			&request.RequestID,
			&request.UserID,
			&request.Username,
			&request.Nome,
			&request.Cognome,
			&request.Email,
			&request.ProfilePic,
			&createdAt,
			&request.Status,
		)
		if err != nil {
			return nil, err
		}

		request.RequestDate = createdAt.Format("2006-01-02 15:04:05")
		requests = append(requests, request)
	}

	return requests, rows.Err()
}

// CancelFriendRequest - Annulla una richiesta di amicizia inviata
func (r *FriendRepository) CancelFriendRequest(requestID, senderID int64) error {
	// Verifica che l'utente sia il mittente della richiesta
	var actualSenderID int64
	err := r.db.QueryRow(`
		SELECT sender_id FROM friend_requests 
		WHERE id = $1 AND status = 'pending'`, requestID).Scan(&actualSenderID)

	if err != nil {
		return fmt.Errorf("richiesta non trovata o non più pendente: %v", err)
	}

	if actualSenderID != senderID {
		return fmt.Errorf("non autorizzato ad annullare questa richiesta")
	}

	// Aggiorna lo status della richiesta
	_, err = r.db.Exec(`
		UPDATE friend_requests 
		SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1`, requestID)

	return err
}

// SearchUsers - Cerca utenti per username o email (escludendo se stesso e già amici)
func (r *FriendRepository) SearchUsers(searchTerm string, currentUserID int64) ([]models.UserSearchResult, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT
			u.id,
			u.username,
			u.nome,
			u.cognome,
			u.email,
			u.profile_picture
		FROM users u
		WHERE 
			(LOWER(u.username) LIKE LOWER($1) OR LOWER(u.email) LIKE LOWER($1))
			AND u.id != $2
			AND u.id NOT IN (
				-- Escludi utenti già amici
				SELECT 
					CASE 
						WHEN f.user1_id = $2 THEN f.user2_id
						ELSE f.user1_id
					END
				FROM friendships f
				WHERE f.user1_id = $2 OR f.user2_id = $2
			)
			AND u.id NOT IN (
				-- Escludi utenti con richieste pendenti (inviate o ricevute)
				SELECT fr.receiver_id FROM friend_requests fr 
				WHERE fr.sender_id = $2 AND fr.status = 'pending'
				UNION
				SELECT fr.sender_id FROM friend_requests fr 
				WHERE fr.receiver_id = $2 AND fr.status = 'pending'
			)
		ORDER BY u.username
		LIMIT 20`,
		"%"+searchTerm+"%", currentUserID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.UserSearchResult
	for rows.Next() {
		var user models.UserSearchResult

		err := rows.Scan(
			&user.UserID,
			&user.Username,
			&user.Nome,
			&user.Cognome,
			&user.Email,
			&user.ProfilePic,
		)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, rows.Err()
}

// GetLatestFriendRequestID ottiene l'ID della richiesta più recente
func (r *FriendRepository) GetLatestFriendRequestID(senderID, receiverID int64) (int64, error) {
	var requestID int64
	query := `
		SELECT id FROM friend_requests 
		WHERE sender_id = $1 AND receiver_id = $2 AND status = 'pending'
		ORDER BY created_at DESC 
		LIMIT 1`

	err := r.db.QueryRow(query, senderID, receiverID).Scan(&requestID)
	if err != nil {
		return 0, err
	}
	return requestID, nil
}

// GetFriendRequestReceiver ottiene l'ID del destinatario di una richiesta di amicizia
func (r *FriendRepository) GetFriendRequestReceiver(requestID int64) (int64, error) {
	var receiverID int64
	query := `SELECT receiver_id FROM friend_requests WHERE id = $1`

	err := r.db.QueryRow(query, requestID).Scan(&receiverID)
	if err != nil {
		return 0, err
	}
	return receiverID, nil
}

// GetUserUnreadFriendRequestsCount ottiene il numero di richieste di amicizia non lette
func (r *FriendRepository) GetUserUnreadFriendRequestsCount(userID int64) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM friend_requests 
		WHERE receiver_id = $1 AND status = 'pending'`

	err := r.db.QueryRow(query, userID).Scan(&count)
	return count, err
}