package repositories

import (
	"database/sql"
	"fmt"

	"trovagiocatoriAuth/internal/models"
)

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// CreateNotification crea una nuova notifica
func (r *NotificationRepository) CreateNotification(notification *models.Notification) error {
	query := `
		INSERT INTO notifications (user_id, type, title, message, status, related_id, sender_id, expires_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(
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
func (r *NotificationRepository) GetUserNotifications(userID int64, limit int, offset int) ([]models.Notification, error) {
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

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		var senderInfo models.SenderInfo

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
func (r *NotificationRepository) GetUnreadNotificationsCount(userID int64) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM notifications 
		WHERE user_id = $1 AND status = 'unread'
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)`

	err := r.db.QueryRow(query, userID).Scan(&count)
	return count, err
}

// GetNotificationsSummary ottiene un riassunto delle notifiche dell'utente
func (r *NotificationRepository) GetNotificationsSummary(userID int64) (*models.NotificationSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_unread,
			COUNT(CASE WHEN type = 'friend_request' THEN 1 END) as friend_requests,
			COUNT(CASE WHEN type = 'event_invite' THEN 1 END) as event_invites,
			COUNT(CASE WHEN type = 'post_comment' THEN 1 END) as comments
		FROM notifications 
		WHERE user_id = $1 AND status = 'unread'
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)`

	var summary models.NotificationSummary
	err := r.db.QueryRow(query, userID).Scan(
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
func (r *NotificationRepository) MarkNotificationAsRead(notificationID, userID int64) error {
	query := `
		UPDATE notifications 
		SET status = 'read', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(query, notificationID, userID)
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
func (r *NotificationRepository) MarkAllNotificationsAsRead(userID int64) error {
	query := `
		UPDATE notifications 
		SET status = 'read', updated_at = CURRENT_TIMESTAMP 
		WHERE user_id = $1 AND status = 'unread'`

	_, err := r.db.Exec(query, userID)
	return err
}

// DeleteNotification elimina una notifica
func (r *NotificationRepository) DeleteNotification(notificationID, userID int64) error {
	query := `DELETE FROM notifications WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(query, notificationID, userID)
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
func (r *NotificationRepository) DeleteExpiredNotifications() error {
	query := `DELETE FROM notifications WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP`
	_, err := r.db.Exec(query)
	return err
}

// DeleteNotificationByRelated elimina notifiche correlate
func (r *NotificationRepository) DeleteNotificationByRelated(userID int64, notificationType models.NotificationType, relatedID int64) error {
	query := `DELETE FROM notifications WHERE user_id = $1 AND type = $2 AND related_id = $3`
	_, err := r.db.Exec(query, userID, notificationType, relatedID)
	return err
}

// CreateFriendRequestNotification crea una notifica per richiesta di amicizia
func (r *NotificationRepository) CreateFriendRequestNotification(receiverID, senderID, requestID int64, senderUsername string) error {
	notification := &models.Notification{
		UserID:    receiverID,
		Type:      models.NotificationTypeFriendRequest,
		Title:     "Nuova richiesta di amicizia",
		Message:   fmt.Sprintf("%s ti ha inviato una richiesta di amicizia", senderUsername),
		Status:    models.NotificationStatusUnread,
		RelatedID: &requestID,
		SenderID:  &senderID,
	}

	return r.CreateNotification(notification)
}

// CreateEventInviteNotification crea una notifica per invito evento
func (r *NotificationRepository) CreateEventInviteNotification(receiverID, senderID, postID int64, senderUsername, eventTitle string) error {
	notification := &models.Notification{
		UserID:    receiverID,
		Type:      models.NotificationTypeEventInvite,
		Title:     "Invito a evento",
		Message:   fmt.Sprintf("%s ti ha invitato all'evento: %s", senderUsername, eventTitle),
		Status:    models.NotificationStatusUnread,
		RelatedID: &postID,
		SenderID:  &senderID,
	}

	return r.CreateNotification(notification)
}

// GetNotificationStats ottiene statistiche sulle notifiche per il cleanup service
func (r *NotificationRepository) GetNotificationStats() (map[string]int, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'unread' THEN 1 END) as unread,
			COUNT(CASE WHEN status = 'read' THEN 1 END) as read,
			COUNT(CASE WHEN type = 'friend_request' THEN 1 END) as friend_requests,
			COUNT(CASE WHEN type = 'event_invite' THEN 1 END) as event_invites,
			COUNT(CASE WHEN expires_at IS NOT NULL AND expires_at < NOW() THEN 1 END) as expired
		FROM notifications`

	var total, unread, read, friendRequests, eventInvites, expired int
	err := r.db.QueryRow(query).Scan(&total, &unread, &read, &friendRequests, &eventInvites, &expired)
	if err != nil {
		return nil, err
	}

	return map[string]int{
		"total":           total,
		"unread":          unread,
		"read":            read,
		"friend_requests": friendRequests,
		"event_invites":   eventInvites,
		"expired":         expired,
	}, nil
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