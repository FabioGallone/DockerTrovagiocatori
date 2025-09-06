// auth-service/internal/handlers/notifications.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/sessions"
)

// NotificationResponse rappresenta la risposta per le operazioni sulle notifiche
type NotificationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// GetNotificationsHandler restituisce le notifiche dell'utente
func GetNotificationsHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica autenticazione
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unauthorized: session_id non presente", http.StatusUnauthorized)
			return
		}

		userID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Parametri di paginazione
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		limit := 20 // Default
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		offset := 0 // Default
		if offsetStr != "" {
			if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		// Ottieni le notifiche
		notifications, err := database.GetUserNotifications(userID, limit, offset)
		if err != nil {
			http.Error(w, "Errore durante il recupero delle notifiche", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := NotificationResponse{
			Success: true,
			Data: map[string]interface{}{
				"notifications": notifications,
				"count":        len(notifications),
				"limit":        limit,
				"offset":       offset,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetNotificationsSummaryHandler restituisce un riassunto delle notifiche
func GetNotificationsSummaryHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica autenticazione
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unauthorized: session_id non presente", http.StatusUnauthorized)
			return
		}

		userID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Ottieni il riassunto delle notifiche
		summary, err := database.GetNotificationsSummary(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero del riassunto notifiche", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := NotificationResponse{
			Success: true,
			Data:    summary,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// MarkNotificationAsReadHandler segna una notifica come letta
func MarkNotificationAsReadHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica autenticazione
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unauthorized: session_id non presente", http.StatusUnauthorized)
			return
		}

		userID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Ottieni l'ID della notifica dall'URL
		notificationIDStr := r.URL.Query().Get("id")
		if notificationIDStr == "" {
			http.Error(w, "ID notifica mancante", http.StatusBadRequest)
			return
		}

		notificationID, err := strconv.ParseInt(notificationIDStr, 10, 64)
		if err != nil {
			http.Error(w, "ID notifica non valido", http.StatusBadRequest)
			return
		}

		// Segna come letta
		err = database.MarkNotificationAsRead(notificationID, userID)
		if err != nil {
			if err.Error() == "notifica non trovata o non autorizzata" {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, "Errore durante l'aggiornamento della notifica", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := NotificationResponse{
			Success: true,
			Message: "Notifica segnata come letta",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// MarkAllNotificationsAsReadHandler segna tutte le notifiche come lette
func MarkAllNotificationsAsReadHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica autenticazione
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unauthorized: session_id non presente", http.StatusUnauthorized)
			return
		}

		userID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Segna tutte come lette
		err = database.MarkAllNotificationsAsRead(userID)
		if err != nil {
			http.Error(w, "Errore durante l'aggiornamento delle notifiche", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := NotificationResponse{
			Success: true,
			Message: "Tutte le notifiche sono state segnate come lette",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// DeleteNotificationHandler elimina una notifica
func DeleteNotificationHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica autenticazione
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unauthorized: session_id non presente", http.StatusUnauthorized)
			return
		}

		userID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Ottieni l'ID della notifica dall'URL
		notificationIDStr := r.URL.Query().Get("id")
		if notificationIDStr == "" {
			http.Error(w, "ID notifica mancante", http.StatusBadRequest)
			return
		}

		notificationID, err := strconv.ParseInt(notificationIDStr, 10, 64)
		if err != nil {
			http.Error(w, "ID notifica non valido", http.StatusBadRequest)
			return
		}

		// Elimina la notifica
		err = database.DeleteNotification(notificationID, userID)
		if err != nil {
			if err.Error() == "notifica non trovata o non autorizzata" {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, "Errore durante l'eliminazione della notifica", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := NotificationResponse{
			Success: true,
			Message: "Notifica eliminata con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// NotificationTestHandler - Endpoint per testare le notifiche (solo per sviluppo)
func NotificationTestHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica autenticazione
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unauthorized: session_id non presente", http.StatusUnauthorized)
			return
		}

		userID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Crea una notifica di test
		notification := &db.Notification{
			UserID:  userID,
			Type:    db.NotificationTypeGeneral,
			Title:   "Notifica di Test",
			Message: "Questa è una notifica di test per verificare il sistema",
			Status:  db.NotificationStatusUnread,
		}

		err = database.CreateNotification(notification)
		if err != nil {
			http.Error(w, "Errore durante la creazione della notifica di test", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := NotificationResponse{
			Success: true,
			Message: "Notifica di test creata con successo",
			Data:    notification,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}