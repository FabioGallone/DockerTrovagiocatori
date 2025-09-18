package middleware

import (
	"fmt"
	"net/http"

	"trovagiocatoriAuth/internal/database/repositories"
	"trovagiocatoriAuth/internal/sessions"
)

// RequireAuth è un middleware per verificare l'autenticazione
func RequireAuth(sm *sessions.SessionManager) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				http.Error(w, "Unauthorized: session_id non presente", http.StatusUnauthorized)
				return
			}

			_, err = sm.GetUserIDBySessionID(cookie.Value)
			if err != nil {
				http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
				return
			}

			next(w, r)
		}
	}
}

// RequireAdmin middleware per verificare privilegi admin
func RequireAdmin(userRepo *repositories.UserRepository, sm *sessions.SessionManager) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				fmt.Printf("[ADMIN] No session cookie found\n")
				http.Error(w, "Unauthorized: session_id non presente", http.StatusUnauthorized)
				return
			}

			userID, err := sm.GetUserIDBySessionID(cookie.Value)
			if err != nil {
				fmt.Printf("[ADMIN] Invalid session: %v\n", err)
				http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
				return
			}

			isAdmin, err := userRepo.CheckUserIsAdmin(userID)
			if err != nil {
				fmt.Printf("[ADMIN] Error checking admin status for userID %d: %v\n", userID, err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !isAdmin {
				fmt.Printf("[ADMIN] Access denied for userID %d: not an admin\n", userID)
				http.Error(w, "Forbidden: privilegi amministratore richiesti", http.StatusForbidden)
				return
			}

			fmt.Printf("[ADMIN] Access granted for admin userID %d\n", userID)
			next(w, r)
		}
	}
}

// GetUserIDFromSession helper per ottenere l'ID utente dalla sessione
func GetUserIDFromSession(r *http.Request, sm *sessions.SessionManager) (int64, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0, err
	}

	return sm.GetUserIDBySessionID(cookie.Value)
}