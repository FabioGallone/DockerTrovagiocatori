package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"trovagiocatoriAuth/internal/database/repositories"
	"trovagiocatoriAuth/internal/middleware"
	"trovagiocatoriAuth/internal/models"
	"trovagiocatoriAuth/internal/sessions"
)

type EventHandler struct {
	eventRepo        *repositories.EventRepository
	userRepo         *repositories.UserRepository
	notificationRepo *repositories.NotificationRepository
	sm               *sessions.SessionManager
}

func NewEventHandler(eventRepo *repositories.EventRepository, userRepo *repositories.UserRepository, sm *sessions.SessionManager) *EventHandler {
	return &EventHandler{
		eventRepo: eventRepo,
		userRepo:  userRepo,
		sm:        sm,
	}
}

// ========== ENDPOINT PREFERITI ==========

type FavoriteRequest struct {
	PostID int `json:"post_id"`
}

type FavoriteResponse struct {
	Success    bool `json:"success"`
	IsFavorite bool `json:"is_favorite"`
}

// AddFavoriteHandler aggiunge un post ai preferiti
func (h *EventHandler) AddFavoriteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		var req FavoriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		if err := h.eventRepo.AddFavorite(userID, req.PostID); err != nil {
			http.Error(w, "Errore durante l'aggiunta ai preferiti", http.StatusInternalServerError)
			return
		}

		response := FavoriteResponse{
			Success:    true,
			IsFavorite: true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// RemoveFavoriteHandler rimuove un post dai preferiti
func (h *EventHandler) RemoveFavoriteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		var req FavoriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		if err := h.eventRepo.RemoveFavorite(userID, req.PostID); err != nil {
			http.Error(w, "Errore durante la rimozione dai preferiti", http.StatusInternalServerError)
			return
		}

		response := FavoriteResponse{
			Success:    true,
			IsFavorite: false,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// CheckFavoriteHandler controlla se un post è nei preferiti
func (h *EventHandler) CheckFavoriteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {


		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Estrai post_id dall'URL
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 3 {
			http.Error(w, "Post ID mancante nell'URL", http.StatusBadRequest)
			return
		}

		postID, err := strconv.Atoi(pathParts[len(pathParts)-1])  //Atoi sta per ASCII to integer
		if err != nil {
			http.Error(w, "Post ID non valido", http.StatusBadRequest)
			return
		}

		isFavorite, err := h.eventRepo.IsFavorite(userID, postID)
		if err != nil {
			http.Error(w, "Errore durante la verifica preferiti", http.StatusInternalServerError)
			return
		}

		response := FavoriteResponse{
			Success:    true,
			IsFavorite: isFavorite,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetUserFavoritesHandler restituisce tutti i preferiti dell'utente
func (h *EventHandler) GetUserFavoritesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		favorites, err := h.eventRepo.GetUserFavorites(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero dei preferiti", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"favorites": favorites,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// ========== ENDPOINT PARTECIPAZIONE EVENTI ==========

type ParticipationRequest struct {
	PostID int `json:"post_id"`
}

type ParticipationResponse struct {
	Success       bool   `json:"success"`
	IsParticipant bool   `json:"is_participant"`
	Message       string `json:"message,omitempty"`
}

// JoinEventHandler - Iscrive l'utente a un evento
func (h *EventHandler) JoinEventHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		var req ParticipationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		if err := h.eventRepo.JoinEvent(userID, req.PostID); err != nil {
			if strings.Contains(err.Error(), "già iscritto") {
				response := ParticipationResponse{
					Success:       false,
					IsParticipant: true,
					Message:       "Sei già iscritto a questo evento",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}
			http.Error(w, "Errore durante l'iscrizione all'evento", http.StatusInternalServerError)
			return
		}

		response := ParticipationResponse{
			Success:       true,
			IsParticipant: true,
			Message:       "Iscrizione all'evento avvenuta con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// LeaveEventHandler - Disiscrive l'utente da un evento
func (h *EventHandler) LeaveEventHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		var req ParticipationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		if err := h.eventRepo.LeaveEvent(userID, req.PostID); err != nil {
			http.Error(w, "Errore durante la disiscrizione dall'evento", http.StatusInternalServerError)
			return
		}

		response := ParticipationResponse{
			Success:       true,
			IsParticipant: false,
			Message:       "Disiscrizione dall'evento avvenuta con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// CheckParticipationHandler - Controlla se l'utente è iscritto a un evento
func (h *EventHandler) CheckParticipationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Estrai post_id dall'URL
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 3 {
			http.Error(w, "Post ID mancante nell'URL", http.StatusBadRequest)
			return
		}

		postID, err := strconv.Atoi(pathParts[len(pathParts)-1])
		if err != nil {
			http.Error(w, "Post ID non valido", http.StatusBadRequest)
			return
		}

		isParticipant, err := h.eventRepo.IsEventParticipant(userID, postID)
		if err != nil {
			http.Error(w, "Errore durante la verifica partecipazione", http.StatusInternalServerError)
			return
		}

		response := ParticipationResponse{
			Success:       true,
			IsParticipant: isParticipant,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetEventParticipantsHandler - Ottiene la lista dei partecipanti a un evento
func (h *EventHandler) GetEventParticipantsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Estrai post_id dall'URL
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 3 {
			http.Error(w, "Post ID mancante nell'URL", http.StatusBadRequest)
			return
		}

		postID, err := strconv.Atoi(pathParts[len(pathParts)-2]) // events/{postID}/participants
		if err != nil {
			http.Error(w, "Post ID non valido", http.StatusBadRequest)
			return
		}

		participants, err := h.eventRepo.GetEventParticipants(postID)
		if err != nil {
			http.Error(w, "Errore durante il recupero dei partecipanti", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success":      true,
			"participants": participants,
			"count":        len(participants),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetUserParticipationsHandler - Ottiene tutti gli eventi a cui l'utente è iscritto
func (h *EventHandler) GetUserParticipationsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		participations, err := h.eventRepo.GetUserParticipations(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero delle partecipazioni", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success":        true,
			"participations": participations,
			"count":          len(participations),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// ========== ENDPOINT INVITI EVENTI ==========

type EventInviteRequest struct {
	PostID      int    `json:"post_id"`
	FriendEmail string `json:"friend_email"`
	Message     string `json:"message"`
}

type EventInviteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// SendEventInviteHandler - Invia un invito per un evento a un amico con notifica
func (h *EventHandler) SendEventInviteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		var req EventInviteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		fmt.Printf("[EVENT_INVITE] Invitation from userID %d to %s for event %d\n", userID, req.FriendEmail, req.PostID)

		// Verifica che l'utente destinatario esista e sia amico
		friendUserID, err := h.userRepo.GetUserIDByEmail(req.FriendEmail)
		if err != nil {
			response := EventInviteResponse{
				Success: false,
				Message: "Utente destinatario non trovato",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		
		// Verifica che non ci sia già un invito pendente
		hasInvite, err := h.eventRepo.CheckPendingEventInvite(friendUserID, req.PostID)
		if err != nil {
			http.Error(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		if hasInvite {
			response := EventInviteResponse{
				Success: false,
				Message: "Invito già inviato a questo utente",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Invia l'invito
		if err := h.eventRepo.SendEventInvite(userID, friendUserID, req.PostID, req.Message); err != nil {
			fmt.Printf("[EVENT_INVITE] Error while sending: %v\n", err)
			http.Error(w, "Errore durante l'invio dell'invito", http.StatusInternalServerError)
			return
		}

		// Ottieni le informazioni del mittente per la notifica
		senderProfile, err := h.userRepo.GetUserProfile(fmt.Sprintf("%d", userID))
		if err != nil {
			fmt.Printf("[EVENT_INVITE] WARNING: Unable to get sender profile: %v\n", err)
		}

		// Ottieni il titolo dell'evento per la notifica
		eventTitle := "Evento sportivo" //fallback
		if postDetails, err := h.eventRepo.GetPostTitleByID(req.PostID); err == nil {
			eventTitle = postDetails
		}

		// Crea la notifica per l'invito evento
		senderUsername := "Utente sconosciuto" //fallback
		if err == nil {
			senderUsername = senderProfile.Username
		}

		if h.notificationRepo != nil {
			notifErr := h.notificationRepo.CreateEventInviteNotification(friendUserID, userID, int64(req.PostID), senderUsername, eventTitle)
			if notifErr != nil {
				fmt.Printf("[EVENT_INVITE] WARNING: Error creating notification: %v\n", notifErr)
			} else {
				fmt.Printf("[EVENT_INVITE] Notification created for event invitation from %d to %d\n", userID, friendUserID)

			}
		}

		fmt.Printf("[EVENT_INVITE] Invitation sent successfully\n")

		response := EventInviteResponse{
			Success: true,
			Message: "Invito inviato con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetEventInvitesHandler - Ottiene gli inviti ricevuti dall'utente
func (h *EventHandler) GetEventInvitesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		invites, err := h.eventRepo.GetEventInvites(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero degli inviti", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"invites": invites,
			"count":   len(invites),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// AcceptEventInviteHandler - Accetta un invito per un evento e rimuove la notifica
func (h *EventHandler) AcceptEventInviteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		inviteIDStr := r.URL.Query().Get("invite_id")
		if inviteIDStr == "" {
			http.Error(w, "invite_id mancante", http.StatusBadRequest)
			return
		}

		inviteID, err := strconv.ParseInt(inviteIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invite_id non valido", http.StatusBadRequest)
			return
		}

		// Ottieni i dettagli dell'invito prima di accettarlo per la notifica
		postID, getPostIDErr := h.eventRepo.GetEventInvitePostID(inviteID)

		// Accetta l'invito (questo iscriverà automaticamente l'utente all'evento)
		if err := h.eventRepo.AcceptEventInvite(inviteID, userID); err != nil {
			http.Error(w, "Errore durante l'accettazione dell'invito", http.StatusInternalServerError)
			return
		}

		// Rimuovi la notifica correlata quando l'invito viene accettato
		if getPostIDErr == nil && h.notificationRepo != nil {
			notifErr := h.notificationRepo.DeleteNotificationByRelated(userID, models.NotificationTypeEventInvite, postID)
			if notifErr != nil {
				fmt.Printf("[EVENT_INVITE] WARNING: Error removing notification after acceptance: %v\n", notifErr)
			} else {
				fmt.Printf("[EVENT_INVITE] Notification removed after accepting invite %d\n", inviteID)

			}
		}

		response := EventInviteResponse{
			Success: true,
			Message: "Invito accettato! Sei ora iscritto all'evento",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// RejectEventInviteHandler - Rifiuta un invito per un evento e rimuove la notifica
func (h *EventHandler) RejectEventInviteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		inviteIDStr := r.URL.Query().Get("invite_id")
		if inviteIDStr == "" {
			http.Error(w, "invite_id mancante", http.StatusBadRequest)
			return
		}

		inviteID, err := strconv.ParseInt(inviteIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invite_id non valido", http.StatusBadRequest)
			return
		}

		// Ottieni i dettagli dell'invito prima di rifiutarlo per la notifica
		postID, getPostIDErr := h.eventRepo.GetEventInvitePostID(inviteID)

		// Rifiuta l'invito
		if err := h.eventRepo.RejectEventInvite(inviteID, userID); err != nil {
			http.Error(w, "Errore durante il rifiuto dell'invito", http.StatusInternalServerError)
			return
		}

		// Rimuovi la notifica correlata quando l'invito viene rifiutato
		if getPostIDErr == nil && h.notificationRepo != nil {
			notifErr := h.notificationRepo.DeleteNotificationByRelated(userID, models.NotificationTypeEventInvite, postID)
			if notifErr != nil {
				fmt.Printf("[EVENT_INVITE] WARNING: Error removing notification after rejection: %v\n", notifErr)
			} else {
				fmt.Printf("[EVENT_INVITE] Notification removed after rejecting invite %d\n", inviteID)

			}
		}

		response := EventInviteResponse{
			Success: true,
			Message: "Invito rifiutato",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetAvailableFriendsForInviteHandler - Ottiene gli amici disponibili per essere invitati a un evento
func (h *EventHandler) GetAvailableFriendsForInviteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		postIDStr := r.URL.Query().Get("post_id")
		if postIDStr == "" {
			http.Error(w, "post_id mancante", http.StatusBadRequest)
			return
		}

		postID, err := strconv.Atoi(postIDStr)
		if err != nil {
			http.Error(w, "post_id non valido", http.StatusBadRequest)
			return
		}

		fmt.Printf("[AVAILABLE_FRIENDS] Searching available friends for user %d and post %d\n", userID, postID)

		availableFriends, err := h.eventRepo.GetAvailableFriendsForInvite(userID, postID)
		if err != nil {
			fmt.Printf("[AVAILABLE_FRIENDS] Error retrieving available friends: %v\n", err)
			http.Error(w, "Errore durante il recupero degli amici disponibili", http.StatusInternalServerError)
			return
		}

		fmt.Printf("[AVAILABLE_FRIENDS] Found %d available friends for the invite\n", len(availableFriends))

		response := map[string]interface{}{
			"success":           true,
			"available_friends": availableFriends,
			"count":             len(availableFriends),
			"post_id":           postID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}