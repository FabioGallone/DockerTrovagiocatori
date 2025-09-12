// auth-service/internal/handlers/bans.go
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

// BanResponse rappresenta la risposta per le operazioni sui ban
type BanResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// BanUserHandler banna un utente (richiesta POST)
func BanUserHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[BAN] Richiesta ban utente\n")

		// Ottieni admin ID dalla sessione
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		adminID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Decodifica la richiesta
		var banReq db.BanUserRequest
		if err := json.NewDecoder(r.Body).Decode(&banReq); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Validazioni
		if banReq.UserID == 0 {
			http.Error(w, "user_id mancante", http.StatusBadRequest)
			return
		}

		if banReq.UserID == adminID {
			http.Error(w, "Non puoi bannare te stesso", http.StatusBadRequest)
			return
		}

		if banReq.Reason == "" {
			banReq.Reason = "Ban amministrativo"
		}

		if banReq.BanType == "" {
			banReq.BanType = "temporary"
		}

		// Ottieni IP per audit
		ipAddress := getClientIP(r)
		userAgent := r.UserAgent()

		// Esegui il ban
		ban, err := database.BanUser(&banReq, adminID, ipAddress, userAgent)
		if err != nil {
			fmt.Printf("[BAN] Errore nel ban utente %d: %v\n", banReq.UserID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("[BAN] ✅ Utente %d bannato con successo da admin %d\n", banReq.UserID, adminID)

		// Risposta di successo
		response := BanResponse{
			Success: true,
			Message: "Utente bannato con successo",
			Data:    ban,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// UnbanUserHandler rimuove il ban di un utente
func UnbanUserHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
		// Estrai user_id dall'URL
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 4 {
			http.Error(w, "User ID mancante nell'URL", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(pathParts[len(pathParts)-1], 10, 64)
		if err != nil {
			http.Error(w, "User ID non valido", http.StatusBadRequest)
			return
		}

		// Ottieni admin ID dalla sessione
		cookie, _ := r.Cookie("session_id")
		adminID, _ := sm.GetUserIDBySessionID(cookie.Value)

		fmt.Printf("[BAN] Tentativo rimozione ban utente %d da admin %d\n", userID, adminID)

		// Rimuovi il ban
		err = database.UnbanUser(userID, adminID, "Ban rimosso dall'amministratore")
		if err != nil {
			fmt.Printf("[BAN] Errore rimozione ban utente %d: %v\n", userID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("[BAN] ✅ Ban rimosso per utente %d da admin %d\n", userID, adminID)

		response := BanResponse{
			Success: true,
			Message: "Ban rimosso con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// GetActiveBansHandler ottiene tutti i ban attivi
func GetActiveBansHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[BAN] Richiesta lista ban attivi\n")

		bans, err := database.GetAllActiveBans()
		if err != nil {
			fmt.Printf("[BAN] Errore recupero ban attivi: %v\n", err)
			http.Error(w, "Errore nel recupero dei ban attivi", http.StatusInternalServerError)
			return
		}

		response := BanResponse{
			Success: true,
			Data:    bans,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// GetUserBanHandler ottiene informazioni sui ban di un utente specifico
func GetUserBanHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
		// Estrai user_id dall'URL
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 5 {
			http.Error(w, "User ID mancante nell'URL", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(pathParts[len(pathParts)-1], 10, 64)
		if err != nil {
			http.Error(w, "User ID non valido", http.StatusBadRequest)
			return
		}

		// Controlla se l'utente è bannato
		isBanned, ban, err := database.IsUserBanned(userID)
		if err != nil {
			fmt.Printf("[BAN] Errore controllo ban utente %d: %v\n", userID, err)
			http.Error(w, "Errore nel controllo del ban", http.StatusInternalServerError)
			return
		}

		response := BanResponse{
			Success: true,
			Data: map[string]interface{}{
				"user_id":   userID,
				"is_banned": isBanned,
				"ban_info":  ban,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// GetUserBanHistoryHandler ottiene la cronologia dei ban di un utente
func GetUserBanHistoryHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
		// Estrai user_id dall'URL
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 5 {
			http.Error(w, "User ID mancante nell'URL", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(pathParts[len(pathParts)-1], 10, 64)
		if err != nil {
			http.Error(w, "User ID non valido", http.StatusBadRequest)
			return
		}

		history, err := database.GetUserBanHistory(userID)
		if err != nil {
			fmt.Printf("[BAN] Errore recupero cronologia ban utente %d: %v\n", userID, err)
			http.Error(w, "Errore nel recupero della cronologia", http.StatusInternalServerError)
			return
		}

		response := BanResponse{
			Success: true,
			Data:    history,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// GetBanStatsHandler ottiene statistiche sui ban
func GetBanStatsHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return requireAdmin(database, sm, func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[BAN] Richiesta statistiche ban\n")

		stats, err := database.GetBanStats()
		if err != nil {
			fmt.Printf("[BAN] Errore recupero statistiche: %v\n", err)
			http.Error(w, "Errore nel recupero delle statistiche", http.StatusInternalServerError)
			return
		}

		response := BanResponse{
			Success: true,
			Data:    stats,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// Helper function per ottenere l'IP del client
func getClientIP(r *http.Request) string {
	// Controlla gli header standard per proxy
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Prendi solo il primo IP se ce ne sono multipli
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fallback sull'IP remoto
	ip := r.RemoteAddr
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}
	return ip
}
