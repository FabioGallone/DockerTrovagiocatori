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
			"success":    true,
			"message":    "Commento eliminato con successo",
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

// AdminToggleUserStatusHandler ora gestisce BAN/UNBAN invece di semplice attiva/disattiva
func AdminToggleUserStatusHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
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
		cookie, _ := r.Cookie("session_id")
		adminID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			http.Error(w, "Sessione non valida", http.StatusUnauthorized)
			return
		}

		// Verifica che non si stia bannando se stesso
		if adminID == targetUserID {
			http.Error(w, "Non puoi bannare/sbannare te stesso", http.StatusBadRequest)
			return
		}

		// Verifica che non si stia bannando un altro admin
		isTargetAdmin, err := database.CheckUserIsAdmin(targetUserID)
		if err != nil {
			fmt.Printf("[ADMIN] Errore controllo admin status: %v\n", err)
			http.Error(w, "Errore nel controllo privilegi", http.StatusInternalServerError)
			return
		}

		if isTargetAdmin {
			http.Error(w, "Non puoi bannare un altro amministratore", http.StatusBadRequest)
			return
		}

		// Controlla lo stato attuale dell'utente
		isBanned, _, err := database.IsUserBanned(targetUserID)
		if err != nil {
			fmt.Printf("[ADMIN] Errore controllo ban utente %d: %v\n", targetUserID, err)
			http.Error(w, "Errore nel controllo dello status utente", http.StatusInternalServerError)
			return
		}

		if isBanned {
			// L'utente è bannato -> SBAN
			fmt.Printf("[ADMIN] Rimozione ban per utente %d da admin %d\n", targetUserID, adminID)

			err = database.UnbanUser(targetUserID, adminID, "Ban rimosso dall'amministratore via pannello admin")
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
				"new_status": true, // L'utente è ora attivo
				"action":     "unbanned",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			// L'utente NON è bannato -> BAN
			fmt.Printf("[ADMIN] Applicazione ban per utente %d da admin %d\n", targetUserID, adminID)

			// Crea una richiesta di ban standard
			banReq := &db.BanUserRequest{
				UserID:  targetUserID,
				Reason:  "Ban amministrativo tramite pannello admin",
				BanType: "temporary", // Ban temporaneo di default
				Notes:   "Banned via admin toggle",
			}

			// Ottieni IP e User-Agent per audit
			ipAddress := getClientIP(r)
			userAgent := r.UserAgent()

			// Applica il ban
			ban, err := database.BanUser(banReq, adminID, ipAddress, userAgent)
			if err != nil {
				fmt.Printf("[ADMIN] Errore applicazione ban utente %d: %v\n", targetUserID, err)
				http.Error(w, "Errore nell'applicazione del ban", http.StatusInternalServerError)
				return
			}

			fmt.Printf("[ADMIN] ✅ Utente %d bannato con successo\n", targetUserID)

			response := map[string]interface{}{
				"success":    true,
				"message":    "Utente bannato con successo",
				"user_id":    targetUserID,
				"is_banned":  true,
				"new_status": false, // L'utente è ora inattivo
				"ban_info":   ban,
				"action":     "banned",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
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
