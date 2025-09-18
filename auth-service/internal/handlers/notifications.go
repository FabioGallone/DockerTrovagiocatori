package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"trovagiocatoriAuth/internal/database/repositories"
	"trovagiocatoriAuth/internal/middleware"
	"trovagiocatoriAuth/internal/models"
	"trovagiocatoriAuth/internal/sessions"
)

type NotificationHandler struct {
	notificationRepo *repositories.NotificationRepository
	sm               *sessions.SessionManager
}

func NewNotificationHandler(notificationRepo *repositories.NotificationRepository, sm *sessions.SessionManager) *NotificationHandler {
	return &NotificationHandler{
		notificationRepo: notificationRepo,
		sm:               sm,
	}
}

// NotificationResponse rappresenta la risposta per le operazioni sulle notifiche
type NotificationResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// GetNotificationsHandler restituisce le notifiche dell'utente
func (h *NotificationHandler) GetNotificationsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Parametri di paginazione
		limitStr := r.URL.Query().Get("limit") // Quanti elementi restituire per pagina
		offsetStr := r.URL.Query().Get("offset") // Quanti elementi saltare dall'inizio

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
		notifications, err := h.notificationRepo.GetUserNotifications(userID, limit, offset)
		if err != nil {
			http.Error(w, "Errore durante il recupero delle notifiche", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := NotificationResponse{
			Success: true,
			Data: map[string]interface{}{
				"notifications": notifications,
				"count":         len(notifications),
				"limit":         limit,
				"offset":        offset,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetNotificationsSummaryHandler restituisce un riassunto delle notifiche
func (h *NotificationHandler) GetNotificationsSummaryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Ottieni il riassunto delle notifiche
		summary, err := h.notificationRepo.GetNotificationsSummary(userID)
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
func (h *NotificationHandler) MarkNotificationAsReadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
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
		err = h.notificationRepo.MarkNotificationAsRead(notificationID, userID)
		if err != nil {
			if err.Error() == "notification not found or not authorized" {
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
func (h *NotificationHandler) MarkAllNotificationsAsReadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Segna tutte come lette
		err = h.notificationRepo.MarkAllNotificationsAsRead(userID)
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
func (h *NotificationHandler) DeleteNotificationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
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
		err = h.notificationRepo.DeleteNotification(notificationID, userID)
		if err != nil {
			if err.Error() == "notification not found or not authorized" {
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

