// auth-service/internal/db/friends.go
package db

import (
	"fmt"
	"time"
)

// CreateFriendsTablesIfNotExists - Crea le tabelle per gli amici se non esistono
func (db *Database) CreateFriendsTablesIfNotExists() error {
	// Tabella delle amicizie
	_, err := db.Conn.Exec(`
	CREATE TABLE IF NOT EXISTS friendships (
		id SERIAL PRIMARY KEY,
		user1_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		user2_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user1_id, user2_id),
		CHECK(user1_id != user2_id),
		CHECK(user1_id < user2_id)
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella friendships: %v", err)
	}

	// Tabella delle richieste di amicizia
	_, err = db.Conn.Exec(`
	CREATE TABLE IF NOT EXISTS friend_requests (
		id SERIAL PRIMARY KEY,
		sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		receiver_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(sender_id, receiver_id),
		CHECK(sender_id != receiver_id),
		CHECK(status IN ('pending', 'accepted', 'rejected', 'cancelled'))
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella friend_requests: %v", err)
	}

	// Indici per migliorare le performance
	_, err = db.Conn.Exec(`
	CREATE INDEX IF NOT EXISTS idx_friendships_user1 ON friendships(user1_id);
	CREATE INDEX IF NOT EXISTS idx_friendships_user2 ON friendships(user2_id);
	CREATE INDEX IF NOT EXISTS idx_friend_requests_sender ON friend_requests(sender_id);
	CREATE INDEX IF NOT EXISTS idx_friend_requests_receiver ON friend_requests(receiver_id);
	CREATE INDEX IF NOT EXISTS idx_friend_requests_status ON friend_requests(status);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione degli indici: %v", err)
	}

	fmt.Println("✔ Tabelle degli amici create/verificate con successo!")
	return nil
}

// GetUserIDByEmail - Ottiene l'ID utente tramite email
func (db *Database) GetUserIDByEmail(email string) (int64, error) {
	var userID int64
	err := db.Conn.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

// SendFriendRequest - Invia una richiesta di amicizia
func (db *Database) SendFriendRequest(senderID, receiverID int64) error {
	_, err := db.Conn.Exec(`
		INSERT INTO friend_requests (sender_id, receiver_id, status) 
		VALUES ($1, $2, 'pending')
		ON CONFLICT (sender_id, receiver_id) 
		DO UPDATE SET status = 'pending', updated_at = CURRENT_TIMESTAMP`,
		senderID, receiverID)
	return err
}

// CheckPendingFriendRequest - Controlla se esiste una richiesta di amicizia pendente
func (db *Database) CheckPendingFriendRequest(userID1, userID2 int64) (bool, error) {
	var count int
	err := db.Conn.QueryRow(`
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
func (db *Database) CheckFriendship(userID1, userID2 int64) (bool, error) {
	// Assicurati che user1_id < user2_id per la query
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	var count int
	err := db.Conn.QueryRow(`
		SELECT COUNT(*) FROM friendships 
		WHERE user1_id = $1 AND user2_id = $2`,
		userID1, userID2).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// AcceptFriendRequest - Accetta una richiesta di amicizia
func (db *Database) AcceptFriendRequest(requestID, receiverID int64) error {
	tx, err := db.Conn.Begin()
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

	// Crea l'amicizia (assicurati che user1_id < user2_id)
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
func (db *Database) RejectFriendRequest(requestID, receiverID int64) error {
	// Verifica che l'utente sia il destinatario della richiesta
	var actualReceiverID int64
	err := db.Conn.QueryRow(`
		SELECT receiver_id FROM friend_requests 
		WHERE id = $1 AND status = 'pending'`, requestID).Scan(&actualReceiverID)

	if err != nil {
		return fmt.Errorf("richiesta non trovata o non più pendente: %v", err)
	}

	if actualReceiverID != receiverID {
		return fmt.Errorf("non autorizzato a rifiutare questa richiesta")
	}

	// Aggiorna lo status della richiesta
	_, err = db.Conn.Exec(`
		UPDATE friend_requests 
		SET status = 'rejected', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1`, requestID)

	return err
}

// RemoveFriendship - Rimuove un'amicizia
func (db *Database) RemoveFriendship(userID1, userID2 int64) error {
	// Assicurati che user1_id < user2_id per la query
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	_, err := db.Conn.Exec(`
		DELETE FROM friendships 
		WHERE user1_id = $1 AND user2_id = $2`,
		userID1, userID2)

	return err
}

// GetFriendsList - Ottiene la lista degli amici di un utente
func (db *Database) GetFriendsList(userID int64) ([]FriendInfo, error) {
	rows, err := db.Conn.Query(`
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

	var friends []FriendInfo
	for rows.Next() {
		var friend FriendInfo
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
func (db *Database) GetFriendRequests(userID int64) ([]FriendRequestInfo, error) {
	rows, err := db.Conn.Query(`
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

	var requests []FriendRequestInfo
	for rows.Next() {
		var request FriendRequestInfo
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
func (db *Database) GetSentFriendRequests(userID int64) ([]FriendRequestInfo, error) {
	rows, err := db.Conn.Query(`
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

	var requests []FriendRequestInfo
	for rows.Next() {
		var request FriendRequestInfo
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
func (db *Database) CancelFriendRequest(requestID, senderID int64) error {
	// Verifica che l'utente sia il mittente della richiesta
	var actualSenderID int64
	err := db.Conn.QueryRow(`
		SELECT sender_id FROM friend_requests 
		WHERE id = $1 AND status = 'pending'`, requestID).Scan(&actualSenderID)

	if err != nil {
		return fmt.Errorf("richiesta non trovata o non più pendente: %v", err)
	}

	if actualSenderID != senderID {
		return fmt.Errorf("non autorizzato ad annullare questa richiesta")
	}

	// Aggiorna lo status della richiesta
	_, err = db.Conn.Exec(`
		UPDATE friend_requests 
		SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1`, requestID)

	return err
}

// SearchUsers - Cerca utenti per username o email (escludendo se stesso e già amici)
func (db *Database) SearchUsers(searchTerm string, currentUserID int64) ([]UserSearchResult, error) {
	rows, err := db.Conn.Query(`
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

	var users []UserSearchResult
	for rows.Next() {
		var user UserSearchResult

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

// GetMutualFriends - Ottiene gli amici in comune tra due utenti
func (db *Database) GetMutualFriends(userID1, userID2 int64) ([]FriendInfo, error) {
	rows, err := db.Conn.Query(`
		SELECT DISTINCT
			u.id,
			u.username,
			u.nome,
			u.cognome,
			u.email,
			u.profile_picture
		FROM users u
		WHERE u.id IN (
			-- Amici del primo utente
			SELECT 
				CASE 
					WHEN f1.user1_id = $1 THEN f1.user2_id
					ELSE f1.user1_id
				END as friend_id
			FROM friendships f1
			WHERE f1.user1_id = $1 OR f1.user2_id = $1
		)
		AND u.id IN (
			-- Amici del secondo utente
			SELECT 
				CASE 
					WHEN f2.user1_id = $2 THEN f2.user2_id
					ELSE f2.user1_id
				END as friend_id
			FROM friendships f2
			WHERE f2.user1_id = $2 OR f2.user2_id = $2
		)
		ORDER BY u.username`,
		userID1, userID2)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mutualFriends []FriendInfo
	for rows.Next() {
		var friend FriendInfo

		err := rows.Scan(
			&friend.UserID,
			&friend.Username,
			&friend.Nome,
			&friend.Cognome,
			&friend.Email,
			&friend.ProfilePic,
		)
		if err != nil {
			return nil, err
		}

		mutualFriends = append(mutualFriends, friend)
	}

	return mutualFriends, rows.Err()
}

// GetFriendsCount - Ottiene il numero di amici di un utente
func (db *Database) GetFriendsCount(userID int64) (int, error) {
	var count int
	err := db.Conn.QueryRow(`
		SELECT COUNT(*) FROM friendships 
		WHERE user1_id = $1 OR user2_id = $1`,
		userID).Scan(&count)

	return count, err
}

// Strutture dati per le funzioni sopra

type FriendInfo struct {
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	Nome         string `json:"nome"`
	Cognome      string `json:"cognome"`
	Email        string `json:"email"`
	ProfilePic   string `json:"profile_picture"`
	FriendsSince string `json:"friends_since"`
}

type FriendRequestInfo struct {
	RequestID   int64  `json:"request_id"`
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	Nome        string `json:"nome"`
	Cognome     string `json:"cognome"`
	Email       string `json:"email"`
	ProfilePic  string `json:"profile_picture"`
	RequestDate string `json:"request_date"`
	Status      string `json:"status"`
}

type UserSearchResult struct {
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
	Nome       string `json:"nome"`
	Cognome    string `json:"cognome"`
	Email      string `json:"email"`
	ProfilePic string `json:"profile_picture"`
}
