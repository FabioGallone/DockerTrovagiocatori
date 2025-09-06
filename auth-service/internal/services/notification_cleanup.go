// auth-service/internal/services/notification_cleanup.go
package services

import (
	"fmt"
	"log"
	"time"

	"trovagiocatoriAuth/internal/db"
)

// NotificationCleanupService gestisce la pulizia delle notifiche scadute
type NotificationCleanupService struct {
	database *db.Database
	ticker   *time.Ticker
	done     chan bool
}

// NewNotificationCleanupService crea un nuovo servizio di pulizia notifiche
func NewNotificationCleanupService(database *db.Database) *NotificationCleanupService {
	return &NotificationCleanupService{
		database: database,
		done:     make(chan bool),
	}
}

// Start avvia il servizio di pulizia che gira ogni ora
func (ncs *NotificationCleanupService) Start() {
	// Esegui pulizia immediata all'avvio
	ncs.cleanupExpiredNotifications()

	// Configura ticker per ogni ora
	ncs.ticker = time.NewTicker(1 * time.Hour)

	go func() {
		for {
			select {
			case <-ncs.ticker.C:
				ncs.cleanupExpiredNotifications()
			case <-ncs.done:
				return
			}
		}
	}()

	log.Println("🧹 Servizio pulizia notifiche avviato (ogni ora)")
}

// Stop ferma il servizio di pulizia
func (ncs *NotificationCleanupService) Stop() {
	if ncs.ticker != nil {
		ncs.ticker.Stop()
	}
	ncs.done <- true
	log.Println("🛑 Servizio pulizia notifiche fermato")
}

// cleanupExpiredNotifications pulisce le notifiche scadute
func (ncs *NotificationCleanupService) cleanupExpiredNotifications() {
	startTime := time.Now()
	
	err := ncs.database.DeleteExpiredNotifications()
	if err != nil {
		log.Printf("❌ Errore durante la pulizia delle notifiche scadute: %v", err)
		return
	}

	duration := time.Since(startTime)
	log.Printf("✅ Pulizia notifiche scadute completata in %v", duration)
}

// CleanupOldReadNotifications pulisce le notifiche lette più vecchie di X giorni
func (ncs *NotificationCleanupService) CleanupOldReadNotifications(daysOld int) error {
	query := `
		DELETE FROM notifications 
		WHERE status = 'read' 
		AND updated_at < NOW() - INTERVAL '%d days'`

	result, err := ncs.database.Conn.Exec(fmt.Sprintf(query, daysOld))
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	log.Printf("🗑️ Eliminate %d notifiche lette più vecchie di %d giorni", rowsAffected, daysOld)
	return nil
}

// GetNotificationStats restituisce statistiche sulle notifiche
func (ncs *NotificationCleanupService) GetNotificationStats() (*NotificationStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'unread' THEN 1 END) as unread,
			COUNT(CASE WHEN status = 'read' THEN 1 END) as read,
			COUNT(CASE WHEN type = 'friend_request' THEN 1 END) as friend_requests,
			COUNT(CASE WHEN type = 'event_invite' THEN 1 END) as event_invites,
			COUNT(CASE WHEN expires_at IS NOT NULL AND expires_at < NOW() THEN 1 END) as expired
		FROM notifications`

	var stats NotificationStats
	err := ncs.database.Conn.QueryRow(query).Scan(
		&stats.Total,
		&stats.Unread,
		&stats.Read,
		&stats.FriendRequests,
		&stats.EventInvites,
		&stats.Expired,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// NotificationStats contiene statistiche sulle notifiche del sistema
type NotificationStats struct {
	Total          int `json:"total"`
	Unread         int `json:"unread"`
	Read           int `json:"read"`
	FriendRequests int `json:"friend_requests"`
	EventInvites   int `json:"event_invites"`
	Expired        int `json:"expired"`
}

// PrintStats stampa le statistiche delle notifiche nei log
func (ncs *NotificationCleanupService) PrintStats() {
	stats, err := ncs.GetNotificationStats()
	if err != nil {
		log.Printf("❌ Errore nel recupero statistiche notifiche: %v", err)
		return
	}

	log.Printf("📊 STATISTICHE NOTIFICHE:")
	log.Printf("   Total: %d | Unread: %d | Read: %d", stats.Total, stats.Unread, stats.Read)
	log.Printf("   Friend Requests: %d | Event Invites: %d | Expired: %d", 
		stats.FriendRequests, stats.EventInvites, stats.Expired)
}