// auth-service/internal/handlers/event_invites.go
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/sessions"
)

type EventInviteRequest struct {
	PostID      int    `json:"post_id"`
	FriendEmail string `json:"friend_email"`
	Message     string `json:"message"`
}

type EventInviteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// SendEventInviteHandler - Invia un invito per un evento a un amico
func SendEventInviteHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Decodifica la richiesta
		var req EventInviteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		fmt.Printf("[EVENT_INVITE] Invito da userID %d a %s per evento %d\n", userID, req.FriendEmail, req.PostID)

		// Verifica che l'utente destinatario esista e sia amico
		friendUserID, err := database.GetUserIDByEmail(req.FriendEmail)
		if err != nil {
			response := EventInviteResponse{
				Success: false,
				Message: "Utente destinatario non trovato",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Verifica che siano amici
		isFriend, err := database.CheckFriendship(userID, friendUserID)
		if err != nil || !isFriend {
			response := EventInviteResponse{
				Success: false,
				Message: "Puoi invitare solo i tuoi amici",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Verifica che non ci sia già un invito pendente
		hasInvite, err := database.CheckPendingEventInvite(friendUserID, req.PostID)
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
		if err := database.SendEventInvite(userID, friendUserID, req.PostID, req.Message); err != nil {
			fmt.Printf("[EVENT_INVITE] Errore nell'invio: %v\n", err)
			http.Error(w, "Errore durante l'invio dell'invito", http.StatusInternalServerError)
			return
		}

		fmt.Printf("[EVENT_INVITE] ✅ Invito inviato con successo\n")

		// Risposta di successo
		response := EventInviteResponse{
			Success: true,
			Message: "Invito inviato con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetEventInvitesHandler - Ottiene gli inviti ricevuti dall'utente
func GetEventInvitesHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni gli inviti ricevuti
		invites, err := database.GetEventInvites(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero degli inviti", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := map[string]interface{}{
			"success": true,
			"invites": invites,
			"count":   len(invites),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// AcceptEventInviteHandler - Accetta un invito per un evento
func AcceptEventInviteHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni invite_id dall'URL
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

		// Accetta l'invito (questo iscriverà automaticamente l'utente all'evento)
		if err := database.AcceptEventInvite(inviteID, userID); err != nil {
			http.Error(w, "Errore durante l'accettazione dell'invito", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := EventInviteResponse{
			Success: true,
			Message: "Invito accettato! Sei ora iscritto all'evento",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// RejectEventInviteHandler - Rifiuta un invito per un evento
func RejectEventInviteHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni invite_id dall'URL
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

		// Rifiuta l'invito
		if err := database.RejectEventInvite(inviteID, userID); err != nil {
			http.Error(w, "Errore durante il rifiuto dell'invito", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := EventInviteResponse{
			Success: true,
			Message: "Invito rifiutato",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
