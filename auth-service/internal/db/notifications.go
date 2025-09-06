// auth-service/internal/db/notifications.go
package db

import (
	"fmt"
	"time"
)

// NotificationType enum per i tipi di notifica
type NotificationType string

const (
	NotificationTypeFriendRequest NotificationType = "friend_request"
	NotificationTypeEventInvite   NotificationType = "event_invite"
	NotificationTypePostComment   NotificationType = "post_comment"
	NotificationTypeGeneral       NotificationType = "general"
)

// NotificationStatus enum per lo stato della notifica
type NotificationStatus string

const (
	NotificationStatusUnread NotificationStatus = "unread"
	NotificationStatusRead   NotificationStatus = "read"
)

// Notification rappresenta una notifica nel sistema
type Notification struct {
	ID           int64              `json:"id" db:"id"`
	UserID       int64              `json:"user_id" db:"user_id"`
	Type         NotificationType   `json:"type" db:"type"`
	Title        string             `json:"title" db:"title"`
	Message      string             `json:"message" db:"message"`
	Status       NotificationStatus `json:"status" db:"status"`
	RelatedID    *int64             `json:"related_id,omitempty" db:"related_id"` // ID della richiesta/evento correlato
	SenderID     *int64             `json:"sender_id,omitempty" db:"sender_id"`   // ID dell'utente che ha scatenato la notifica
	SenderInfo   *SenderInfo        `json:"sender_info,omitempty"`               // Informazioni del mittente (populate manually)
	CreatedAt    time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" db:"updated_at"`
	ExpiresAt    *time.Time         `json:"expires_at,omitempty" db:"expires_at"`
}

// SenderInfo contiene le informazioni del mittente della notifica
type SenderInfo struct {
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	Nome         string `json:"nome"`
	Cognome      string `json:"cognome"`
	Email        string `json:"email"`
	ProfilePic   string `json:"profile_picture"`
	DisplayName  string `json:"display_name"`
}

// NotificationSummary rappresenta un riassunto delle notifiche dell'utente
type NotificationSummary struct {
	UnreadCount      int `json:"unread_count"`
	FriendRequests   int `json:"friend_requests"`
	EventInvites     int `json:"event_invites"`
	Comments         int `json:"comments"`
	HasNotifications bool `json:"has_notifications"`
}

// CreateNotificationsTableIfNotExists crea la tabella delle notifiche se non esiste
func (db *Database) CreateNotificationsTableIfNotExists() error {
	_, err := db.Conn.Exec(`
	CREATE TABLE IF NOT EXISTS notifications (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		type VARCHAR(50) NOT NULL,
		title VARCHAR(255) NOT NULL,
		message TEXT NOT NULL,
		status VARCHAR(20) DEFAULT 'unread',
		related_id INTEGER,
		sender_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP,
		CHECK(status IN ('unread', 'read')),
		CHECK(type IN ('friend_request', 'event_invite', 'post_comment', 'general'))
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella notifications: %v", err)
	}

	// Indici per migliorare le performance
	_, err = db.Conn.Exec(`
	CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
	CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
	CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type);
	CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
	CREATE INDEX IF NOT EXISTS idx_notifications_user_status ON notifications(user_id, status);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione degli indici notifications: %v", err)
	}

	fmt.Println("✔ Tabella notifications creata/verificata con successo!")
	return nil
}

// CreateNotification crea una nuova notifica
func (db *Database) CreateNotification(notification *Notification) error {
	query := `
		INSERT INTO notifications (user_id, type, title, message, status, related_id, sender_id, expires_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	err := db.Conn.QueryRow(
		query,
		notification.UserID,
		notification.Type,
		notification.Title,
		notification.Message,
		notification.Status,
		notification.RelatedID,
		notification.SenderID,
		notification.ExpiresAt,
	).Scan(&notification.ID, &notification.CreatedAt, &notification.UpdatedAt)

	return err
}

// GetUserNotifications ottiene tutte le notifiche di un utente
func (db *Database) GetUserNotifications(userID int64, limit int, offset int) ([]Notification, error) {
	query := `
		SELECT 
			n.id, n.user_id, n.type, n.title, n.message, n.status, 
			n.related_id, n.sender_id, n.created_at, n.updated_at, n.expires_at,
			u.username, u.nome, u.cognome, u.email, u.profile_picture
		FROM notifications n
		LEFT JOIN users u ON n.sender_id = u.id
		WHERE n.user_id = $1 
		AND (n.expires_at IS NULL OR n.expires_at > CURRENT_TIMESTAMP)
		ORDER BY n.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := db.Conn.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		var senderInfo SenderInfo

		err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.Status,
			&n.RelatedID, &n.SenderID, &n.CreatedAt, &n.UpdatedAt, &n.ExpiresAt,
			&senderInfo.Username, &senderInfo.Nome, &senderInfo.Cognome,
			&senderInfo.Email, &senderInfo.ProfilePic,
		)
		if err != nil {
			return nil, err
		}

		// Aggiungi le informazioni del mittente se presenti
		if n.SenderID != nil {
			senderInfo.UserID = *n.SenderID
			senderInfo.DisplayName = getDisplayName(senderInfo.Username, senderInfo.Nome, senderInfo.Cognome)
			n.SenderInfo = &senderInfo
		}

		notifications = append(notifications, n)
	}

	return notifications, rows.Err()
}

// GetUnreadNotificationsCount ottiene il numero di notifiche non lette
func (db *Database) GetUnreadNotificationsCount(userID int64) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM notifications 
		WHERE user_id = $1 AND status = 'unread'
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)`

	err := db.Conn.QueryRow(query, userID).Scan(&count)
	return count, err
}

// GetNotificationsSummary ottiene un riassunto delle notifiche dell'utente
func (db *Database) GetNotificationsSummary(userID int64) (*NotificationSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_unread,
			COUNT(CASE WHEN type = 'friend_request' THEN 1 END) as friend_requests,
			COUNT(CASE WHEN type = 'event_invite' THEN 1 END) as event_invites,
			COUNT(CASE WHEN type = 'post_comment' THEN 1 END) as comments
		FROM notifications 
		WHERE user_id = $1 AND status = 'unread'
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)`

	var summary NotificationSummary
	err := db.Conn.QueryRow(query, userID).Scan(
		&summary.UnreadCount,
		&summary.FriendRequests,
		&summary.EventInvites,
		&summary.Comments,
	)

	if err != nil {
		return nil, err
	}

	summary.HasNotifications = summary.UnreadCount > 0
	return &summary, nil
}

// MarkNotificationAsRead segna una notifica come letta
func (db *Database) MarkNotificationAsRead(notificationID, userID int64) error {
	query := `
		UPDATE notifications 
		SET status = 'read', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1 AND user_id = $2`

	result, err := db.Conn.Exec(query, notificationID, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notifica non trovata o non autorizzata")
	}

	return nil
}

// MarkAllNotificationsAsRead segna tutte le notifiche di un utente come lette
func (db *Database) MarkAllNotificationsAsRead(userID int64) error {
	query := `
		UPDATE notifications 
		SET status = 'read', updated_at = CURRENT_TIMESTAMP 
		WHERE user_id = $1 AND status = 'unread'`

	_, err := db.Conn.Exec(query, userID)
	return err
}

// DeleteNotification elimina una notifica
func (db *Database) DeleteNotification(notificationID, userID int64) error {
	query := `DELETE FROM notifications WHERE id = $1 AND user_id = $2`

	result, err := db.Conn.Exec(query, notificationID, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notifica non trovata o non autorizzata")
	}

	return nil
}

// DeleteExpiredNotifications elimina le notifiche scadute
func (db *Database) DeleteExpiredNotifications() error {
	query := `DELETE FROM notifications WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP`
	_, err := db.Conn.Exec(query)
	return err
}

// DeleteNotificationByRelated elimina notifiche correlate (es. quando una richiesta viene accettata)
func (db *Database) DeleteNotificationByRelated(userID int64, notificationType NotificationType, relatedID int64) error {
	query := `DELETE FROM notifications WHERE user_id = $1 AND type = $2 AND related_id = $3`
	_, err := db.Conn.Exec(query, userID, notificationType, relatedID)
	return err
}

// CreateFriendRequestNotification crea una notifica per richiesta di amicizia
func (db *Database) CreateFriendRequestNotification(receiverID, senderID, requestID int64, senderUsername string) error {
	notification := &Notification{
		UserID:    receiverID,
		Type:      NotificationTypeFriendRequest,
		Title:     "Nuova richiesta di amicizia",
		Message:   fmt.Sprintf("%s ti ha inviato una richiesta di amicizia", senderUsername),
		Status:    NotificationStatusUnread,
		RelatedID: &requestID,
		SenderID:  &senderID,
	}

	return db.CreateNotification(notification)
}

// CreateEventInviteNotification crea una notifica per invito evento
func (db *Database) CreateEventInviteNotification(receiverID, senderID, postID int64, senderUsername, eventTitle string) error {
	notification := &Notification{
		UserID:    receiverID,
		Type:      NotificationTypeEventInvite,
		Title:     "Invito a evento",
		Message:   fmt.Sprintf("%s ti ha invitato all'evento: %s", senderUsername, eventTitle),
		Status:    NotificationStatusUnread,
		RelatedID: &postID,
		SenderID:  &senderID,
	}

	return db.CreateNotification(notification)
}

// Utility function per creare display name
func getDisplayName(username, nome, cognome string) string {
	if username != "" {
		return username
	}
	if nome != "" && cognome != "" {
		return fmt.Sprintf("%s %s", nome, cognome)
	}
	if nome != "" {
		return nome
	}
	return "Utente sconosciuto"
}