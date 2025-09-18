package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"trovagiocatoriAuth/internal/database/repositories"
	"trovagiocatoriAuth/internal/middleware"
	"trovagiocatoriAuth/internal/models"
	"trovagiocatoriAuth/internal/sessions"
)

type FriendHandler struct {
	friendRepo       *repositories.FriendRepository
	userRepo         *repositories.UserRepository
	notificationRepo *repositories.NotificationRepository
	sm               *sessions.SessionManager
}

func NewFriendHandler(friendRepo *repositories.FriendRepository, userRepo *repositories.UserRepository, notificationRepo *repositories.NotificationRepository, sm *sessions.SessionManager) *FriendHandler {
	return &FriendHandler{
		friendRepo:       friendRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		sm:               sm,
	}
}

type FriendRequest struct {
	TargetEmail string `json:"target_email"`
}

type FriendResponse struct {
	Success  bool                       `json:"success"`
	Message  string                     `json:"message,omitempty"`
	IsFriend bool                       `json:"is_friend,omitempty"`
	Friends  []models.FriendInfo        `json:"friends,omitempty"`
	Requests []models.FriendRequestInfo `json:"requests,omitempty"`
}

// SendFriendRequestHandler - Invia una richiesta di amicizia con notifica
func (h *FriendHandler) SendFriendRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			fmt.Printf("[FRIENDS ERROR] Sessione non valida: %v\n", err)
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		var req FriendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fmt.Printf("[FRIENDS ERROR] Formato richiesta non valido: %v\n", err)
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		fmt.Printf("[FRIENDS DEBUG] Friend request from userID %d to email %s\n", userID, req.TargetEmail)

		// Verifica che l'email target esista
		targetUserID, err := h.userRepo.GetUserIDByEmail(req.TargetEmail)
		if err != nil {
			fmt.Printf("[FRIENDS ERROR] Email destinatario non trovata (%s): %v\n", req.TargetEmail, err)
			response := FriendResponse{
				Success: false,
				Message: "Utente non trovato",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Verifica che non siano già amici
		isFriend, err := h.friendRepo.CheckFriendship(userID, targetUserID)
		if err != nil {
			fmt.Printf("[FRIENDS ERROR] Errore controllo amicizia: %v\n", err)
			http.Error(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		if isFriend {
			response := FriendResponse{
				Success: false,
				Message: "Siete già amici",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Verifica che non ci sia già una richiesta pendente
		hasRequest, err := h.friendRepo.CheckPendingFriendRequest(userID, targetUserID)
		if err != nil {
			fmt.Printf("[FRIENDS ERROR] Errore controllo richiesta pendente: %v\n", err)
			http.Error(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		if hasRequest {
			response := FriendResponse{
				Success: false,
				Message: "Richiesta di amicizia già inviata",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Invia la richiesta di amicizia
		if err := h.friendRepo.SendFriendRequest(userID, targetUserID); err != nil {
			fmt.Printf("[FRIENDS ERROR] Errore invio richiesta: %v\n", err)
			http.Error(w, "Errore durante l'invio della richiesta", http.StatusInternalServerError)
			return
		}

		// Crea notifica per il destinatario
		h.createFriendRequestNotification(userID, targetUserID)

		fmt.Printf("[FRIENDS SUCCESS] Friend request successfully sent from %d to %d\n", userID, targetUserID)


		response := FriendResponse{
			Success: true,
			Message: "Richiesta di amicizia inviata con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// AcceptFriendRequestHandler - Accetta una richiesta di amicizia
func (h *FriendHandler) AcceptFriendRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		requestIDStr := r.URL.Query().Get("request_id")
		if requestIDStr == "" {
			http.Error(w, "request_id mancante", http.StatusBadRequest)
			return
		}

		requestID, err := strconv.ParseInt(requestIDStr, 10, 64) //numero decimale covertita in un intero a 64 bit.
		if err != nil {
			http.Error(w, "request_id non valido", http.StatusBadRequest)
			return
		}

		// Accetta la richiesta
		if err := h.friendRepo.AcceptFriendRequest(requestID, userID); err != nil {
			http.Error(w, "Errore durante l'accettazione della richiesta", http.StatusInternalServerError)
			return
		}

		// Rimuovi la notifica correlata
		h.removeNotificationAfterAction(userID, models.NotificationTypeFriendRequest, requestID, "accettazione")

		response := FriendResponse{
			Success: true,
			Message: "Richiesta di amicizia accettata",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// RejectFriendRequestHandler - Rifiuta una richiesta di amicizia
func (h *FriendHandler) RejectFriendRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		requestIDStr := r.URL.Query().Get("request_id")
		if requestIDStr == "" {
			http.Error(w, "request_id mancante", http.StatusBadRequest)
			return
		}

		requestID, err := strconv.ParseInt(requestIDStr, 10, 64)
		if err != nil {
			http.Error(w, "request_id non valido", http.StatusBadRequest)
			return
		}

		// Rifiuta la richiesta
		if err := h.friendRepo.RejectFriendRequest(requestID, userID); err != nil {
			http.Error(w, "Errore durante il rifiuto della richiesta", http.StatusInternalServerError)
			return
		}

		// Rimuovi la notifica correlata
		h.removeNotificationAfterAction(userID, models.NotificationTypeFriendRequest, requestID, "rifiuto")

		response := FriendResponse{
			Success: true,
			Message: "Richiesta di amicizia rifiutata",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// CancelFriendRequestHandler - Annulla una richiesta di amicizia inviata
func (h *FriendHandler) CancelFriendRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		requestIDStr := r.URL.Query().Get("request_id")
		if requestIDStr == "" {
			http.Error(w, "request_id mancante", http.StatusBadRequest)
			return
		}

		requestID, err := strconv.ParseInt(requestIDStr, 10, 64)
		if err != nil {
			http.Error(w, "request_id non valido", http.StatusBadRequest)
			return
		}

		// Ottieni l'ID del destinatario prima di annullare
		receiverID, getReceiverErr := h.friendRepo.GetFriendRequestReceiver(requestID)

		// Annulla la richiesta
		if err := h.friendRepo.CancelFriendRequest(requestID, userID); err != nil {
			http.Error(w, "Errore durante l'annullamento della richiesta", http.StatusInternalServerError)
			return
		}

		// Rimuovi la notifica dal destinatario
		if getReceiverErr == nil {
			h.removeNotificationAfterAction(receiverID, models.NotificationTypeFriendRequest, requestID, "annullamento")
		}

		response := FriendResponse{
			Success: true,
			Message: "Richiesta di amicizia annullata",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// RemoveFriendHandler - Rimuove un'amicizia
func (h *FriendHandler) RemoveFriendHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		var req FriendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Ottieni ID dell'amico da rimuovere
		friendUserID, err := h.userRepo.GetUserIDByEmail(req.TargetEmail)
		if err != nil {
			response := FriendResponse{
				Success: false,
				Message: "Utente non trovato",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Rimuovi l'amicizia
		if err := h.friendRepo.RemoveFriendship(userID, friendUserID); err != nil {
			http.Error(w, "Errore durante la rimozione dell'amicizia", http.StatusInternalServerError)
			return
		}

		response := FriendResponse{
			Success: true,
			Message: "Amicizia rimossa con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// CheckFriendshipHandler - Controlla se due utenti sono amici
func (h *FriendHandler) CheckFriendshipHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		targetEmail := r.URL.Query().Get("email")
		if targetEmail == "" {
			http.Error(w, "email mancante", http.StatusBadRequest)
			return
		}

		// Ottieni ID dell'utente target
		targetUserID, err := h.userRepo.GetUserIDByEmail(targetEmail)
		if err != nil {
			response := FriendResponse{
				Success:  false,
				IsFriend: false,
				Message:  "Utente non trovato",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Controlla l'amicizia
		isFriend, err := h.friendRepo.CheckFriendship(userID, targetUserID)
		if err != nil {
			http.Error(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		response := FriendResponse{
			Success:  true,
			IsFriend: isFriend,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetFriendsListHandler - Ottiene la lista degli amici
func (h *FriendHandler) GetFriendsListHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		friends, err := h.friendRepo.GetFriendsList(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero degli amici", http.StatusInternalServerError)
			return
		}

		response := FriendResponse{
			Success: true,
			Friends: friends,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetFriendRequestsHandler - Ottiene le richieste di amicizia ricevute
func (h *FriendHandler) GetFriendRequestsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		requests, err := h.friendRepo.GetFriendRequests(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero delle richieste", http.StatusInternalServerError)
			return
		}

		response := FriendResponse{
			Success:  true,
			Requests: requests,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetSentFriendRequestsHandler - Ottiene le richieste di amicizia inviate
func (h *FriendHandler) GetSentFriendRequestsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		requests, err := h.friendRepo.GetSentFriendRequests(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero delle richieste inviate", http.StatusInternalServerError)
			return
		}

		response := FriendResponse{
			Success:  true,
			Requests: requests,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// SearchUsersHandler - Cerca utenti per username o email
func (h *FriendHandler) SearchUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		searchTerm := r.URL.Query().Get("q") //friends/search?q=mario
		if searchTerm == "" {
			http.Error(w, "Termine di ricerca mancante", http.StatusBadRequest)
			return
		}

		if len(searchTerm) < 3 {
			http.Error(w, "Il termine di ricerca deve essere di almeno 3 caratteri", http.StatusBadRequest)
			return
		}

		users, err := h.friendRepo.SearchUsers(searchTerm, userID)
		if err != nil {
			http.Error(w, "Errore durante la ricerca utenti", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

// Helper methods

// createFriendRequestNotification crea una notifica per richiesta di amicizia
func (h *FriendHandler) createFriendRequestNotification(senderID, receiverID int64) {
	// Ottieni le informazioni del mittente
	senderProfile, err := h.userRepo.GetUserProfile(fmt.Sprintf("%d", senderID))
	if err != nil {
		fmt.Printf("[FRIENDS WARNING] Unable to get sender profile: %v\n", err)
		return
	}

	// Ottieni l'ID della richiesta appena creata
	requestID, err := h.friendRepo.GetLatestFriendRequestID(senderID, receiverID)
	if err != nil {
		fmt.Printf("[FRIENDS WARNING] Unable to get request ID: %v\n", err)
		return
	}

	// Crea la notifica
	notifErr := h.notificationRepo.CreateFriendRequestNotification(receiverID, senderID, requestID, senderProfile.Username)
	if notifErr != nil {
		fmt.Printf("[FRIENDS WARNING] Error creating notification: %v\n", notifErr)
	} else {
		fmt.Printf("[FRIENDS SUCCESS] ✅ Notification created for friend request from %d to %d\n", senderID, receiverID)
	}
}

// removeNotificationAfterAction rimuove una notifica dopo un'azione
func (h *FriendHandler) removeNotificationAfterAction(userID int64, notifType models.NotificationType, relatedID int64, action string) {
	notifErr := h.notificationRepo.DeleteNotificationByRelated(userID, notifType, relatedID)
	if notifErr != nil {
		fmt.Printf("[FRIENDS WARNING] Error removing notification after %s: %v\n", action, notifErr)
	} else {
		fmt.Printf("[FRIENDS SUCCESS] Notification removed after %s request %d\n", action, relatedID)
	}
}