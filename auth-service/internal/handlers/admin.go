// auth-service/internal/handlers/admin.go
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/sessions"
)

// Middleware per verificare privilegi admin
func requireAdmin(database *db.Database, sm *sessions.SessionManager, next http.HandlerFunc) http.HandlerFunc {
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

		isAdmin, err := database.CheckUserIsAdmin(userID)
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

// AdminDeletePostHandler elimina un post (solo admin)
func AdminDeletePostHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
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

		fmt.Printf("[ADMIN] Tentativo eliminazione post %d\n", postID)

		// Chiama il backend Python per eliminare il post
		pythonURL := "http://backend_python:8000/admin/posts/" + strconv.Itoa(postID)
		
		// Propaga il cookie di sessione
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

		fmt.Printf("[ADMIN] ✅ Post %d eliminato con successo\n", postID)

		// Risposta successo
		response := map[string]interface{}{
			"success": true,
			"message": "Post eliminato con successo",
			"post_id": postID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// AdminDeleteCommentHandler elimina un commento (solo admin)
func AdminDeleteCommentHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
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

		fmt.Printf("[ADMIN] Tentativo eliminazione commento %d\n", commentID)

		// Chiama il backend Python per eliminare il commento
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

		fmt.Printf("[ADMIN] ✅ Commento %d eliminato con successo\n", commentID)

		response := map[string]interface{}{
			"success": true,
			"message": "Commento eliminato con successo",
			"comment_id": commentID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// AdminGetUsersHandler restituisce tutti gli utenti per il pannello admin
func AdminGetUsersHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[ADMIN] Richiesta lista utenti\n")

		users, err := database.GetAllUsers()
		if err != nil {
			fmt.Printf("[ADMIN] Errore nel recupero utenti: %v\n", err)
			http.Error(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		fmt.Printf("[ADMIN] ✅ Recuperati %d utenti\n", len(users))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})
}

// AdminToggleUserStatusHandler attiva/disattiva un utente
func AdminToggleUserStatusHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
		// Estrai user_id dall'URL
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 5 {
			http.Error(w, "User ID mancante nell'URL", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(pathParts[len(pathParts)-2], 10, 64)
		if err != nil {
			http.Error(w, "User ID non valido", http.StatusBadRequest)
			return
		}

		fmt.Printf("[ADMIN] Tentativo toggle status utente %d\n", userID)

		// Verifica che non si stia disattivando se stesso
		cookie, _ := r.Cookie("session_id")
		adminUserID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			http.Error(w, "Sessione non valida", http.StatusUnauthorized)
			return
		}

		if adminUserID == userID {
			http.Error(w, "Non puoi disattivare il tuo stesso account", http.StatusBadRequest)
			return
		}

		// Toggle dello status utente
		newStatus, err := database.ToggleUserStatus(userID)
		if err != nil {
			fmt.Printf("[ADMIN] Errore toggle status utente %d: %v\n", userID, err)
			http.Error(w, "Errore nella modifica dello status utente", http.StatusInternalServerError)
			return
		}

		statusText := "disattivato"
		if newStatus {
			statusText = "riattivato"
		}

		fmt.Printf("[ADMIN] ✅ Utente %d %s con successo\n", userID, statusText)

		response := map[string]interface{}{
			"success":    true,
			"message":    fmt.Sprintf("Utente %s con successo", statusText),
			"user_id":    userID,
			"new_status": newStatus,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// AdminStatsHandler restituisce statistiche per il dashboard admin
func AdminStatsHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[ADMIN] Richiesta statistiche dashboard\n")

		// Ottieni statistiche dal database Go
		totalUsers, err := database.GetTotalUsersCount()
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

		fmt.Printf("[ADMIN] ✅ Statistiche generate: %+v\n", stats)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})
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