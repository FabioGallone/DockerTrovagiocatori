package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"trovagiocatoriAuth/internal/db"
)

// ProfileHandler restituisce i dati del profilo utente
func ProfileHandler(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Estrai l'ID utente dalla URL (esempio: /profile/1)
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			http.Error(w, "ID utente non valido", http.StatusBadRequest)
			return
		}
		userID := parts[2]

		// Recupera il profilo utente dal database
		user, err := database.GetUserProfile(userID)
		if err != nil {
			http.Error(w, "Utente non trovato", http.StatusNotFound)
			return
		}

		// Invia i dati JSON come risposta
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}
