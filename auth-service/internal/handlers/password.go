// handlers/password.go
package handlers

import (
	"encoding/json"
	"net/http"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/sessions"
)

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func UpdatePasswordHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica il cookie di sessione
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Utente non autenticato", http.StatusUnauthorized)
			return
		}

		// Recupera l'userID dalla sessione
		userID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			http.Error(w, "Sessione non valida", http.StatusUnauthorized)
			return
		}

		// Decodifica la richiesta
		var req PasswordChangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Verifica la password corrente
		valid, err := database.VerifyCurrentPassword(userID, req.CurrentPassword)
		if err != nil || !valid {
			http.Error(w, "Password corrente non valida", http.StatusUnauthorized)
			return
		}

		// Aggiorna la password
		if err := database.UpdateUserPassword(userID, req.NewPassword); err != nil {
			http.Error(w, "Errore durante l'aggiornamento", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Password aggiornata con successo"})
	}
}
