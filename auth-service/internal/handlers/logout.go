package handlers

import (
	"net/http"
	"time"

	"trovagiocatoriAuth/internal/sessions"
)

// LogoutHandler invalida la sessione rimuovendola dal SessionManager
// In internal/handlers/auth.go (o in un file separato)
func LogoutHandler(sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Sessione non trovata", http.StatusBadRequest)
			return
		}
		// Rimuovi la sessione dal SessionManager
		sm.DeleteSession(cookie.Value)

		// Elimina il cookie sul client (impostando una data di scadenza passata)
		expiredCookie := &http.Cookie{
			Name:    "session_id",
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
		}
		http.SetCookie(w, expiredCookie)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Logout effettuato con successo"))
	}
}
