package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/sessions"
)

// LoginRequest rappresenta la struttura dei dati inviati dall'utente per il login
type LoginRequest struct {
	EmailOrUsername string `json:"email_or_username"`
	Password        string `json:"password"`
}

// LoginHandler gestisce il login degli utenti
func LoginHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var loginData LoginRequest
		err := json.NewDecoder(r.Body).Decode(&loginData)
		if err != nil {
			http.Error(w, "Dati non validi", http.StatusBadRequest)
			return
		}

		// Verifica le credenziali dell'utente
		userID, err := database.VerifyUser(loginData.EmailOrUsername, loginData.Password)
		if err != nil {
			http.Error(w, "Credenziali errate", http.StatusUnauthorized)
			return
		}

		// Crea una sessione per l'utente autenticato
		sessionID, _ := sm.CreateSession(userID)
		http.SetCookie(w, &http.Cookie{
			Name:  "session_id",
			Value: sessionID,
			Path:  "/",
		})
		fmt.Printf("✔ SessionID: %s\n", sessionID)
		// Risposta di successo
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Login riuscito"})
	}
}
