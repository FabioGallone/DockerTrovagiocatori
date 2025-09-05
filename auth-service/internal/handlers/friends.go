// auth-service/internal/handlers/friends.go
package handlers

import (
	"encoding/json"
	"fmt"
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
			fmt.Printf("[FRIENDS ERROR] Session cookie non trovato: %v\n", err)
			http.Error(w, "Unauthorized: session_id non presente", http.StatusUnauthorized)
			return
		}

		userID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			fmt.Printf("[FRIENDS ERROR] Sessione non valida per cookie %s: %v\n", cookie.Value, err)
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		// Decodifica la richiesta
		var req FriendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fmt.Printf("[FRIENDS ERROR] Formato richiesta non valido: %v\n", err)
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// DEBUG DETTAGLIATO
		fmt.Printf("[FRIENDS DEBUG] === INIZIO RICHIESTA AMICIZIA ===\n")
		fmt.Printf("[FRIENDS DEBUG] UserID mittente: %d\n", userID)
		fmt.Printf("[FRIENDS DEBUG] Email destinatario: %s\n", req.TargetEmail)
		fmt.Printf("[FRIENDS DEBUG] Cookie session: %s\n", cookie.Value)

		// Verifica che l'email target esista
		targetUserID, err := database.GetUserIDByEmail(req.TargetEmail)
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

		fmt.Printf("[FRIENDS DEBUG] UserID destinatario trovato: %d\n", targetUserID)

		// Verifica che non stia tentando di aggiungere se stesso
		if targetUserID == userID {
			fmt.Printf("[FRIENDS ERROR] Tentativo di aggiungere se stesso: userID=%d, targetUserID=%d\n", userID, targetUserID)
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
			fmt.Printf("[FRIENDS ERROR] Errore controllo amicizia: %v\n", err)
			http.Error(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		if isFriend {
			fmt.Printf("[FRIENDS INFO] Utenti già amici: %d e %d\n", userID, targetUserID)
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
			fmt.Printf("[FRIENDS ERROR] Errore controllo richiesta pendente: %v\n", err)
			http.Error(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		if hasRequest {
			fmt.Printf("[FRIENDS INFO] Richiesta già pendente tra %d e %d\n", userID, targetUserID)
			response := FriendResponse{
				Success: false,
				Message: "Richiesta di amicizia già inviata",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Invia la richiesta di amicizia
		fmt.Printf("[FRIENDS DEBUG] Tentativo di invio richiesta...\n")
		if err := database.SendFriendRequest(userID, targetUserID); err != nil {
			fmt.Printf("[FRIENDS ERROR] Errore invio richiesta nel database: %v\n", err)
			http.Error(w, "Errore durante l'invio della richiesta", http.StatusInternalServerError)
			return
		}

		fmt.Printf("[FRIENDS SUCCESS] ✅ Richiesta inviata con successo da %d a %d\n", userID, targetUserID)
		fmt.Printf("[FRIENDS DEBUG] === FINE RICHIESTA AMICIZIA ===\n")

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

// Aggiungi questi handler al file auth-service/internal/handlers/friends.go

// GetSentFriendRequestsHandler - Ottiene le richieste di amicizia inviate
func GetSentFriendRequestsHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni le richieste di amicizia inviate
		requests, err := database.GetSentFriendRequests(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero delle richieste inviate", http.StatusInternalServerError)
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

// SearchUsersHandler - Cerca utenti per username o email
func SearchUsersHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni termine di ricerca dall'URL
		searchTerm := r.URL.Query().Get("q")
		if searchTerm == "" {
			http.Error(w, "Termine di ricerca mancante", http.StatusBadRequest)
			return
		}

		if len(searchTerm) < 3 {
			http.Error(w, "Il termine di ricerca deve essere di almeno 3 caratteri", http.StatusBadRequest)
			return
		}

		// Cerca gli utenti
		users, err := database.SearchUsers(searchTerm, userID)
		if err != nil {
			http.Error(w, "Errore durante la ricerca utenti", http.StatusInternalServerError)
			return
		}

		// Risposta diretta con array di utenti
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

// CancelFriendRequestHandler - Annulla una richiesta di amicizia inviata
func CancelFriendRequestHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Annulla la richiesta
		if err := database.CancelFriendRequest(requestID, userID); err != nil {
			http.Error(w, "Errore durante l'annullamento della richiesta", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := FriendResponse{
			Success: true,
			Message: "Richiesta di amicizia annullata",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
