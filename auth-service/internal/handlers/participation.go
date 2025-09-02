package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/sessions"
)

type ParticipationRequest struct {
	PostID int `json:"post_id"`
}

type ParticipationResponse struct {
	Success       bool   `json:"success"`
	IsParticipant bool   `json:"is_participant"`
	Message       string `json:"message,omitempty"`
}

type ParticipantInfo struct {
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	Nome         string `json:"nome"`
	Cognome      string `json:"cognome"`
	Email        string `json:"email"`
	ProfilePic   string `json:"profile_picture"`
	RegisteredAt string `json:"registered_at"`
}

// JoinEventHandler - Iscrive l'utente a un evento
func JoinEventHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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
		var req ParticipationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Iscrive l'utente all'evento
		if err := database.JoinEvent(userID, req.PostID); err != nil {
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

		// Risposta di successo
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
func LeaveEventHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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
		var req ParticipationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Disiscrive l'utente dall'evento
		if err := database.LeaveEvent(userID, req.PostID); err != nil {
			http.Error(w, "Errore durante la disiscrizione dall'evento", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
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
func CheckParticipationHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Controlla se è iscritto
		isParticipant, err := database.IsEventParticipant(userID, postID)
		if err != nil {
			http.Error(w, "Errore durante la verifica partecipazione", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := ParticipationResponse{
			Success:       true,
			IsParticipant: isParticipant,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetEventParticipantsHandler - Ottiene la lista dei partecipanti a un evento
func GetEventParticipantsHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni i partecipanti
		participants, err := database.GetEventParticipants(postID)
		if err != nil {
			http.Error(w, "Errore durante il recupero dei partecipanti", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := map[string]interface{}{
			"success":      true,
			"participants": participants,
			"count":        len(participants),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}