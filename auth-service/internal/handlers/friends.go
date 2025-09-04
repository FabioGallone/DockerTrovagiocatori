// auth-service/internal/handlers/friends.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/sessions"
)

type FriendRequest struct {
	TargetEmail string `json:"target_email"`
}

type FriendResponse struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message,omitempty"`
	IsFriend bool                   `json:"is_friend,omitempty"`
	Friends  []db.FriendInfo        `json:"friends,omitempty"`
	Requests []db.FriendRequestInfo `json:"requests,omitempty"`
}

// SendFriendRequestHandler - Invia una richiesta di amicizia
func SendFriendRequestHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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
		var req FriendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Verifica che l'email target esista
		targetUserID, err := database.GetUserIDByEmail(req.TargetEmail)
		if err != nil {
			response := FriendResponse{
				Success: false,
				Message: "Utente non trovato",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Verifica che non stia tentando di aggiungere se stesso
		if targetUserID == userID {
			response := FriendResponse{
				Success: false,
				Message: "Non puoi aggiungere te stesso come amico",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Verifica che non siano già amici
		isFriend, err := database.CheckFriendship(userID, targetUserID)
		if err != nil {
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
		hasRequest, err := database.CheckPendingFriendRequest(userID, targetUserID)
		if err != nil {
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
		if err := database.SendFriendRequest(userID, targetUserID); err != nil {
			http.Error(w, "Errore durante l'invio della richiesta", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := FriendResponse{
			Success: true,
			Message: "Richiesta di amicizia inviata con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// AcceptFriendRequestHandler - Accetta una richiesta di amicizia
func AcceptFriendRequestHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni request_id dall'URL
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

		// Accetta la richiesta
		if err := database.AcceptFriendRequest(requestID, userID); err != nil {
			http.Error(w, "Errore durante l'accettazione della richiesta", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := FriendResponse{
			Success: true,
			Message: "Richiesta di amicizia accettata",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// RejectFriendRequestHandler - Rifiuta una richiesta di amicizia
func RejectFriendRequestHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni request_id dall'URL
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
		if err := database.RejectFriendRequest(requestID, userID); err != nil {
			http.Error(w, "Errore durante il rifiuto della richiesta", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := FriendResponse{
			Success: true,
			Message: "Richiesta di amicizia rifiutata",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// RemoveFriendHandler - Rimuove un'amicizia
func RemoveFriendHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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
		var req FriendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Ottieni ID dell'amico da rimuovere
		friendUserID, err := database.GetUserIDByEmail(req.TargetEmail)
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
		if err := database.RemoveFriendship(userID, friendUserID); err != nil {
			http.Error(w, "Errore durante la rimozione dell'amicizia", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := FriendResponse{
			Success: true,
			Message: "Amicizia rimossa con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// CheckFriendshipHandler - Controlla se due utenti sono amici
func CheckFriendshipHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni email dall'URL
		targetEmail := r.URL.Query().Get("email")
		if targetEmail == "" {
			http.Error(w, "email mancante", http.StatusBadRequest)
			return
		}

		// Ottieni ID dell'utente target
		targetUserID, err := database.GetUserIDByEmail(targetEmail)
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
		isFriend, err := database.CheckFriendship(userID, targetUserID)
		if err != nil {
			http.Error(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := FriendResponse{
			Success:  true,
			IsFriend: isFriend,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetFriendsListHandler - Ottiene la lista degli amici
func GetFriendsListHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni la lista degli amici
		friends, err := database.GetFriendsList(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero degli amici", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := FriendResponse{
			Success: true,
			Friends: friends,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetFriendRequestsHandler - Ottiene le richieste di amicizia ricevute
func GetFriendRequestsHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni le richieste di amicizia
		requests, err := database.GetFriendRequests(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero delle richieste", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := FriendResponse{
			Success:  true,
			Requests: requests,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
