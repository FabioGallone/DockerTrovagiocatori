package services

import (

	"log"
	"time"

	"trovagiocatoriAuth/internal/database/repositories"
)

// NotificationCleanupService gestisce la pulizia delle notifiche scadute
type NotificationCleanupService struct {
	notificationRepo *repositories.NotificationRepository
	ticker           *time.Ticker
	done             chan bool
}

// NewNotificationCleanupService crea un nuovo servizio di pulizia notifiche
func NewNotificationCleanupService(notificationRepo *repositories.NotificationRepository) *NotificationCleanupService {
	return &NotificationCleanupService{
		notificationRepo: notificationRepo,
		done:             make(chan bool),
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
	
	err := ncs.notificationRepo.DeleteExpiredNotifications()
	if err != nil {
		log.Printf("❌ Errore durante la pulizia delle notifiche scadute: %v", err)
		return
	}

	duration := time.Since(startTime)
	log.Printf("✅ Pulizia notifiche scadute completata in %v", duration)
}

// PrintStats stampa le statistiche delle notifiche nei log
func (ncs *NotificationCleanupService) PrintStats() {
	stats, err := ncs.notificationRepo.GetNotificationStats()
	if err != nil {
		log.Printf("❌ Errore nel recupero statistiche notifiche: %v", err)
		return
	}

	log.Printf("📊 STATISTICHE NOTIFICHE:")
	log.Printf("   Total: %d | Unread: %d | Read: %d", 
		stats["total"], stats["unread"], stats["read"])
	log.Printf("   Friend Requests: %d | Event Invites: %d | Expired: %d", 
		stats["friend_requests"], stats["event_invites"], stats["expired"])
}