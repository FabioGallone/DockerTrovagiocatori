package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"trovagiocatoriAuth/internal/database/repositories"
	"trovagiocatoriAuth/internal/middleware"
	"trovagiocatoriAuth/internal/models"
	"trovagiocatoriAuth/internal/sessions"
)

type BanHandler struct {
	banRepo  *repositories.BanRepository
	userRepo *repositories.UserRepository
	sm       *sessions.SessionManager
}

func NewBanHandler(banRepo *repositories.BanRepository, userRepo *repositories.UserRepository, sm *sessions.SessionManager) *BanHandler {
	return &BanHandler{
		banRepo:  banRepo,
		userRepo: userRepo,
		sm:       sm,
	}
}

// BanResponse rappresenta la risposta per le operazioni sui ban
type BanResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// BanUserHandler banna un utente 
func (h *BanHandler) BanUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[BAN] Richiesta ban utente\n")

		// Ottieni admin ID dalla sessione
		adminID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Decodifica la richiesta
		var banReq models.BanUserRequest
		if err := json.NewDecoder(r.Body).Decode(&banReq); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Validazioni
		if banReq.UserID == 0 {
			http.Error(w, "user_id mancante", http.StatusBadRequest)
			return
		}

		// Esegui il ban
		ban, err := h.banRepo.BanUser(&banReq, adminID) 
		if err != nil {
			fmt.Printf("[BAN] Errore nel ban utente %d: %v\n", banReq.UserID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("[BAN] Utente %d bannato con successo da admin %d\n", banReq.UserID, adminID)

		// Risposta di successo
		response := BanResponse{
			Success: true,
			Message: "Utente bannato con successo",
			Data:    ban,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// UnbanUserHandler rimuove il ban di un utente
func (h *BanHandler) UnbanUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		adminID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		fmt.Printf("[BAN] Tentativo rimozione ban utente %d da admin %d\n", userID, adminID)

		// Rimuovi il ban
		err = h.banRepo.UnbanUser(userID, adminID, "Ban rimosso dall'amministratore")
		if err != nil {
			fmt.Printf("[BAN] Errore rimozione ban utente %d: %v\n", userID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("[BAN]  Ban removed for user %d by admin %d\n", userID, adminID)

		response := BanResponse{
			Success: true,
			Message: "Ban rimosso con successo",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetActiveBansHandler ottiene tutti i ban attivi
func (h *BanHandler) GetActiveBansHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[BAN] Requesting list of active bans\n")

		bans, err := h.banRepo.GetAllActiveBans()
		if err != nil {
			fmt.Printf("[BAN] Error retrieving active bans: %v\n", err)
			http.Error(w, "Errore nel recupero dei ban attivi", http.StatusInternalServerError)
			return
		}

		response := BanResponse{
			Success: true,
			Data:    bans,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetUserBanHandler ottiene informazioni sui ban di un utente specifico
func (h *BanHandler) GetUserBanHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		isBanned, ban, err := h.banRepo.IsUserBanned(userID)
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
	}
}

// GetUserBanHistoryHandler ottiene la cronologia dei ban di un utente
func (h *BanHandler) GetUserBanHistoryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		history, err := h.banRepo.GetUserBanHistory(userID)
		if err != nil {
			fmt.Printf("[BAN] Error retrieving ban history for user %d: %v\n", userID, err)
			http.Error(w, "Errore nel recupero della cronologia", http.StatusInternalServerError)
			return
		}

		response := BanResponse{
			Success: true,
			Data:    history,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetBanStatsHandler ottiene statistiche sui ban
func (h *BanHandler) GetBanStatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[BAN] Richiesta statistiche ban\n")

		stats, err := h.banRepo.GetBanStats()
		if err != nil {
			fmt.Printf("[BAN] Error retrieving statistics: %v\n", err)
			http.Error(w, "Errore nel recupero delle statistiche", http.StatusInternalServerError)
			return
		}

		response := BanResponse{
			Success: true,
			Data:    stats,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}