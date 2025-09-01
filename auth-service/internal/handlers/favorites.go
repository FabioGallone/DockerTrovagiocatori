package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/sessions"
)

type FavoriteRequest struct {
	PostID int `json:"post_id"`
}

type FavoriteResponse struct {
	Success    bool `json:"success"`
	IsFavorite bool `json:"is_favorite"`
}

// AddFavoriteHandler aggiunge un post ai preferiti
func AddFavoriteHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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
		var req FavoriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Aggiungi ai preferiti
		if err := database.AddFavorite(userID, req.PostID); err != nil {
			http.Error(w, "Errore durante l'aggiunta ai preferiti", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := FavoriteResponse{
			Success:    true,
			IsFavorite: true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// RemoveFavoriteHandler rimuove un post dai preferiti
func RemoveFavoriteHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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
		var req FavoriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Rimuovi dai preferiti
		if err := database.RemoveFavorite(userID, req.PostID); err != nil {
			http.Error(w, "Errore durante la rimozione dai preferiti", http.StatusInternalServerError)
			return
		}

		// Risposta di successo
		response := FavoriteResponse{
			Success:    true,
			IsFavorite: false,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// CheckFavoriteHandler controlla se un post è nei preferiti
func CheckFavoriteHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Controlla se è nei preferiti
		isFavorite, err := database.IsFavorite(userID, postID)
		if err != nil {
			http.Error(w, "Errore durante la verifica preferiti", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := FavoriteResponse{
			Success:    true,
			IsFavorite: isFavorite,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetUserFavoritesHandler restituisce tutti i preferiti dell'utente
func GetUserFavoritesHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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

		// Ottieni i preferiti dell'utente
		favorites, err := database.GetUserFavorites(userID)
		if err != nil {
			http.Error(w, "Errore durante il recupero dei preferiti", http.StatusInternalServerError)
			return
		}

		// Risposta
		response := map[string]interface{}{
			"success":   true,
			"favorites": favorites,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
