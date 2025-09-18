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

	log.Println("Notification cleanup service started (every hour)")

}

// Stop ferma il servizio di pulizia
func (ncs *NotificationCleanupService) Stop() {
	if ncs.ticker != nil {
		ncs.ticker.Stop()
	}
	ncs.done <- true
	log.Println("Notification cleanup service stopped")

}

// cleanupExpiredNotifications pulisce le notifiche scadute
func (ncs *NotificationCleanupService) cleanupExpiredNotifications() {
	startTime := time.Now()
	
	err := ncs.notificationRepo.DeleteExpiredNotifications()
	if err != nil {
		log.Printf("Error while cleaning up expired notifications: %v", err)

		return
	}

	duration := time.Since(startTime)
	log.Printf("Expired notifications cleanup completed in %v", duration)

}

