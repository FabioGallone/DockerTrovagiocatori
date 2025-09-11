package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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