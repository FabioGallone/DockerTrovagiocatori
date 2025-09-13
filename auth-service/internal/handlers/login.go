// auth-service/internal/handlers/login.go (CORRETTO)
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/sessions"
)

// LoginRequest rappresenta la struttura dei dati inviati dall'utente per il login
type LoginRequest struct {
	EmailOrUsername string `json:"email_or_username"`
	Password        string `json:"password"`
}

// LoginResponse rappresenta la risposta del login
type LoginResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Error   string   `json:"error,omitempty"`
	BanInfo *BanInfo `json:"ban_info,omitempty"`
}

// BanInfo contiene informazioni sul ban per la risposta (SEMPLIFICATO)
type BanInfo struct {
	Reason   string    `json:"reason"`
	BannedAt time.Time `json:"banned_at"`
}

// LoginHandler gestisce il login degli utenti CON CONTROLLO BAN
func LoginHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var loginData LoginRequest
		err := json.NewDecoder(r.Body).Decode(&loginData)
		if err != nil {
			respondWithError(w, "Dati non validi", http.StatusBadRequest)
			return
		}

		fmt.Printf("[LOGIN] Tentativo login per: %s\n", loginData.EmailOrUsername)

		// Verifica le credenziali dell'utente nel db
		userID, err := database.VerifyUser(loginData.EmailOrUsername, loginData.Password)
		if err != nil {
			fmt.Printf("[LOGIN] Credenziali errate per %s: %v\n", loginData.EmailOrUsername, err)
			respondWithError(w, "Credenziali errate", http.StatusUnauthorized)
			return
		}

		fmt.Printf("[LOGIN] Credenziali valide per userID: %d\n", userID)

		// CONTROLLO BAN: Controlla se l'utente è bannato PRIMA di creare la sessione
		isBanned, banInfo, err := database.IsUserBanned(userID)
		if err != nil {
			fmt.Printf("[LOGIN] Errore controllo ban per userID %d: %v\n", userID, err)
			respondWithError(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		if isBanned && banInfo != nil {
			fmt.Printf("[LOGIN] ❌ Accesso negato - Utente %d è bannato: %s\n", userID, banInfo.Reason)

			// Crea la risposta con informazioni sul ban (SEMPLIFICATO)
			banResponse := &BanInfo{
				Reason:   banInfo.Reason,
				BannedAt: banInfo.BannedAt,
			}

			// Messaggio per ban permanente semplificato
			message := "Il tuo account è stato bannato permanentemente."

			// Se c'è un motivo specificato, aggiungilo
			if banInfo.Reason != "" && banInfo.Reason != "Ban amministrativo" {
				message += fmt.Sprintf(" Motivo: %s", banInfo.Reason)
			}

			response := LoginResponse{
				Success: false,
				Error:   message,
				BanInfo: banResponse,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden) // 403 Forbidden per utenti bannati
			json.NewEncoder(w).Encode(response)
			return
		}

		// CONTROLLO ATTIVO: Controlla anche se l'utente è attivo (doppio controllo di sicurezza)
		isActive, err := database.IsUserActive(userID)
		if err != nil {
			fmt.Printf("[LOGIN] Errore controllo stato attivo per userID %d: %v\n", userID, err)
			respondWithError(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		if !isActive {
			fmt.Printf("[LOGIN] ❌ Accesso negato - Utente %d non è attivo\n", userID)
			respondWithError(w, "Account non attivo. Contatta l'amministratore.", http.StatusForbidden)
			return
		}

		// Se tutto OK, procedi con la creazione della sessione
		fmt.Printf("[LOGIN] ✅ Controlli superati, creazione sessione per userID: %d\n", userID)

		// Crea una sessione per l'utente autenticato
		sessionID, err := sm.CreateSession(userID)
		if err != nil {
			fmt.Printf("[LOGIN] Errore creazione sessione per userID %d: %v\n", userID, err)
			respondWithError(w, "Errore nella creazione della sessione", http.StatusInternalServerError)
			return
		}

		// Imposta il cookie
		http.SetCookie(w, &http.Cookie{
			Name:  "session_id",
			Value: sessionID,
			Path:  "/",
		})

		fmt.Printf("[LOGIN] ✅ Login completato con successo per userID %d, SessionID: %s\n", userID, sessionID)

		// Risposta di successo
		response := LoginResponse{
			Success: true,
			Message: "Login riuscito",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// Helper function per rispondere con errore
func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	response := LoginResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
