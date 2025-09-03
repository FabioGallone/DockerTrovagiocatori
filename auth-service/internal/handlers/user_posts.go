package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/sessions"
)

// GetUserEmailHandler - Ottiene l'email dell'utente corrente per il backend Python
func GetUserEmailHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni l'email dell'utente dal database
		user, err := database.GetUserProfile(fmt.Sprintf("%d", userID))
		if err != nil {
			http.Error(w, "Utente non trovato", http.StatusNotFound)
			return
		}

		// Risposta con l'email dell'utente
		response := map[string]interface{}{
			"success":    true,
			"user_email": user.Email,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
