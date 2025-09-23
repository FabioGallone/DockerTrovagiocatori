package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trovagiocatoriAuth/internal/database/repositories"
	"trovagiocatoriAuth/internal/middleware"
	"trovagiocatoriAuth/internal/models"
	"trovagiocatoriAuth/internal/sessions"
)

type AdminHandler struct {
	adminRepo *repositories.AdminRepository
	userRepo  *repositories.UserRepository
	banRepo   *repositories.BanRepository
	sm        *sessions.SessionManager
}

func NewAdminHandler(adminRepo *repositories.AdminRepository, userRepo *repositories.UserRepository, banRepo *repositories.BanRepository, sm *sessions.SessionManager) *AdminHandler {
	return &AdminHandler{
		adminRepo: adminRepo,
		userRepo:  userRepo,
		banRepo:   banRepo,
		sm:        sm,
	}
}

// AdminDeletePostHandler elimina un post (solo admin)
func (h *AdminHandler) AdminDeletePostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Estrai post_id dall'URL
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 4 {
			http.Error(w, "Post ID mancante nell'URL", http.StatusBadRequest)
			return
		}

		postID, err := strconv.Atoi(pathParts[len(pathParts)-1])
		if err != nil {
			http.Error(w, "Post ID non valido", http.StatusBadRequest)
			return
		}

		fmt.Printf("[ADMIN] Attempting to delete post %d\n", postID)

		// Chiama il backend Python per eliminare il post
		pythonURL := "http://backend_python:8000/admin/posts/" + strconv.Itoa(postID) //convertire un intero in una stringa 

		cookie, _ := r.Cookie("session_id")

		// Crea richiesta DELETE al Python backend
		client := &http.Client{}
		req, err := http.NewRequest("DELETE", pythonURL, nil)
		if err != nil {
			fmt.Printf("[ADMIN] Error creating request: %v\n", err)
			http.Error(w, "Errore interno", http.StatusInternalServerError)
			return
		}

		if cookie != nil {
			req.AddCookie(cookie)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[ADMIN] Error calling Python backend: %v\n", err)
			http.Error(w, "Errore comunicazione con backend", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf("[ADMIN] Python backend returned status %d\n", resp.StatusCode)
			http.Error(w, "Errore nell'eliminazione del post", resp.StatusCode)
			return
		}

		fmt.Printf("[ADMIN] Post %d successfully deleted\n", postID)

		// Risposta successo
		response := map[string]interface{}{
			"success": true,
			"message": "Post eliminato con successo",
			"post_id": postID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// AdminDeleteCommentHandler elimina un commento (solo admin)
func (h *AdminHandler) AdminDeleteCommentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Estrai comment_id dall'URL
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 4 {
			http.Error(w, "Comment ID mancante nell'URL", http.StatusBadRequest)
			return
		}

		commentID, err := strconv.Atoi(pathParts[len(pathParts)-1])
		if err != nil {
			http.Error(w, "Comment ID non valido", http.StatusBadRequest)
			return
		}

		fmt.Printf("[ADMIN] Attempting to delete comment %d\n", commentID)

	
		pythonURL := "http://backend_python:8000/admin/comments/" + strconv.Itoa(commentID)

		cookie, _ := r.Cookie("session_id")

		client := &http.Client{}
		req, err := http.NewRequest("DELETE", pythonURL, nil)
		if err != nil {
			fmt.Printf("[ADMIN] Error creating request: %v\n", err)
			http.Error(w, "Errore interno", http.StatusInternalServerError)
			return
		}

		if cookie != nil {
			req.AddCookie(cookie)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[ADMIN] Error calling Python backend: %v\n", err)
			http.Error(w, "Errore comunicazione con backend", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf("[ADMIN] Python backend returned status %d\n", resp.StatusCode)
			http.Error(w, "Errore nell'eliminazione del commento", resp.StatusCode)
			return
		}

		fmt.Printf("[ADMIN] Comment %d successfully deleted\n", commentID)


		response := map[string]interface{}{
			"success":    true,
			"message":    "Commento eliminato con successo",
			"comment_id": commentID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// AdminGetUsersHandler restituisce tutti gli utenti per il pannello admin
func (h *AdminHandler) AdminGetUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[ADMIN] Requesting user list\n")

		users, err := h.adminRepo.GetAllUsers()
		if err != nil {
			fmt.Printf("[ADMIN] Error retrieving users: %v\n", err)
			http.Error(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		fmt.Printf("[ADMIN] Retrieved %d users\n", len(users))


		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

// AdminToggleUserStatusHandler gestisce BAN/UNBAN 
func (h *AdminHandler) AdminToggleUserStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Estrai user_id dall'URL
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 5 {
			http.Error(w, "User ID mancante nell'URL", http.StatusBadRequest)
			return
		}

		targetUserID, err := strconv.ParseInt(pathParts[len(pathParts)-2], 10, 64)
		if err != nil {
			http.Error(w, "User ID non valido", http.StatusBadRequest)
			return
		}

		fmt.Printf("[ADMIN] Toggle ban status per utente %d\n", targetUserID)

		// Ottieni admin ID dalla sessione
		adminID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Sessione non valida", http.StatusUnauthorized)
			return
		}

		// Controlla lo stato attuale dell'utente
		isBanned, _, err := h.banRepo.IsUserBanned(targetUserID)
		if err != nil {
			fmt.Printf("[ADMIN] Errore controllo ban utente %d: %v\n", targetUserID, err)
			http.Error(w, "Errore nel controllo dello status utente", http.StatusInternalServerError)
			return
		}

		if isBanned {
			// L'utente è bannato -> SBAN
			fmt.Printf("[ADMIN] Rimozione ban per utente %d da admin %d\n", targetUserID, adminID)

			err = h.banRepo.UnbanUser(targetUserID, adminID, "Ban rimosso dall'amministratore via pannello admin")
			if err != nil {
				fmt.Printf("[ADMIN] Errore rimozione ban utente %d: %v\n", targetUserID, err)
				http.Error(w, "Errore nella rimozione del ban", http.StatusInternalServerError)
				return
			}

			fmt.Printf("[ADMIN] ✅ Utente %d sbannato con successo\n", targetUserID)

			response := map[string]interface{}{
				"success":    true,
				"message":    "Utente sbannato e riattivato con successo",
				"user_id":    targetUserID,
				"is_banned":  false,
				"new_status": true,
				"action":     "unbanned",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			// L'utente NON è bannato -> BAN PERMANENTE
			fmt.Printf("[ADMIN] Applicazione ban permanente per utente %d da admin %d\n", targetUserID, adminID)

			// Applica il ban permanente
			ban, err := h.banRepo.BanUser(&models.BanUserRequest{
				UserID: targetUserID,
				Notes:  "Banned permanently via admin toggle",
			}, adminID) 
			if err != nil {
				fmt.Printf("[ADMIN] Error applying ban for user %d: %v\n", targetUserID, err)
				http.Error(w, "Errore nell'applicazione del ban", http.StatusInternalServerError)
				return
			}

			fmt.Printf("[ADMIN] User %d permanently banned\n", targetUserID)

			response := map[string]interface{}{
				"success":    true,
				"message":    "Utente bannato permanentemente",
				"user_id":    targetUserID,
				"is_banned":  true,
				"new_status": false,
				"ban_info":   ban,
				"action":     "banned_permanent",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}
}

// AdminStatsHandler restituisce statistiche per il dashboard admin
func (h *AdminHandler) AdminStatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[ADMIN] Requesting dashboard statistics\n")

		// Ottieni statistiche dal database Go
		totalUsers, err := h.adminRepo.GetTotalUsersCount()
		if err != nil {
			fmt.Printf("[ADMIN] Errore nel conteggio utenti: %v\n", err)
			totalUsers = 0
		}

		// Chiama il backend Python per le altre statistiche
		pythonURL := "http://backend_python:8000/admin/stats"

		cookie, _ := r.Cookie("session_id")

		client := &http.Client{}
		req, err := http.NewRequest("GET", pythonURL, nil)
		if err != nil {
			fmt.Printf("[ADMIN] Error creating request to Python: %v\n", err)
			http.Error(w, "Errore interno", http.StatusInternalServerError)
			return
		}

		if cookie != nil {
			req.AddCookie(cookie)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[ADMIN] Error calling Python backend: %v\n", err)
			http.Error(w, "Errore comunicazione con backend", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var pythonStats map[string]interface{}
		if resp.StatusCode == 200 {
			json.NewDecoder(resp.Body).Decode(&pythonStats)
		} else {
			pythonStats = make(map[string]interface{})
		}

		// Combina le statistiche
		stats := map[string]interface{}{
			"total_users":        totalUsers,
			"total_posts":        getIntFromMap(pythonStats, "total_posts", 0),
			"total_comments":     getIntFromMap(pythonStats, "total_comments", 0),
			"total_sport_fields": getIntFromMap(pythonStats, "total_sport_fields", 0),
			"generated_at":       time.Now().Format("2006-01-02 15:04:05"),
		}

		fmt.Printf("[ADMIN] Statistics generated: %+v\n", stats)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

// Helper function per estrarre interi da map
func getIntFromMap(m map[string]interface{}, key string, defaultValue int) int {
	if val, exists := m[key]; exists {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if intVal, err := strconv.Atoi(v); err == nil {
				return intVal
			}
		}
	}
	return defaultValue
}

