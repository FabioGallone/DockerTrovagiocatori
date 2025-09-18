package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"trovagiocatoriAuth/internal/database/repositories"
	"trovagiocatoriAuth/internal/middleware"
	"trovagiocatoriAuth/internal/models"
	"trovagiocatoriAuth/internal/sessions"
	"trovagiocatoriAuth/internal/utils"
)

type AuthHandler struct {
	userRepo *repositories.UserRepository
	banRepo  *repositories.BanRepository
	sm       *sessions.SessionManager
}

func NewAuthHandler(userRepo *repositories.UserRepository, banRepo *repositories.BanRepository, sm *sessions.SessionManager) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		banRepo:  banRepo,
		sm:       sm,
	}
}

// LoginRequest rappresenta la struttura dei dati inviati dall'utente per il login
type LoginRequest struct {
	EmailOrUsername string `json:"email_or_username"`
	Password        string `json:"password"`
}

// LoginResponse rappresenta la risposta del login
type LoginResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Error   string   `json:"error,omitempty"`
	BanInfo *BanInfo `json:"ban_info,omitempty"`
}

// BanInfo contiene informazioni sul ban per la risposta
type BanInfo struct {
	Reason   string    `json:"reason"`
	BannedAt time.Time `json:"banned_at"`
}

// RegisterHandler gestisce la registrazione di un nuovo utente
func (h *AuthHandler) RegisterHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Limita la dimensione massima dell'upload a 5MB
		r.ParseMultipartForm(5 << 20)

		// Recupera i dati dal form
		nome := r.FormValue("nome")
		cognome := r.FormValue("cognome")
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		if nome == "" || cognome == "" || username == "" || email == "" || password == "" {
			http.Error(w, "Tutti i campi sono obbligatori", http.StatusBadRequest)
			return
		}

		// Hash della password
		hashedPassword, err := utils.HashPassword(password)
		if err != nil {
			http.Error(w, "Error hashing the password", http.StatusInternalServerError)
			return
		}

		// Gestione immagine profilo
		var profilePictureFilename string
		file, handler, err := r.FormFile("profile_picture")
		if err == nil {
			defer file.Close()

			uploadDir := "uploads/profile_pictures"
			if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
				if mkdirErr := os.MkdirAll(uploadDir, os.ModePerm); mkdirErr != nil {
					http.Error(w, fmt.Sprintf("Error creating the directory: %v", mkdirErr), http.StatusInternalServerError)
					return
				}
			}

			extension := filepath.Ext(handler.Filename)
			profilePictureFilename = fmt.Sprintf("%s%s", username, extension)
			filePath := filepath.Join(uploadDir, profilePictureFilename)

			outFile, err := os.Create(filePath)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error creating the file: %v", err), http.StatusInternalServerError)
				return
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, file)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error writing the file: %v", err), http.StatusInternalServerError)
				return
			}
			fmt.Printf(" Image successfully saved: %s\n", filePath)

		}

		newUser := models.User{
			Nome:       nome,
			Cognome:    cognome,
			Username:   username,
			Email:      email,
			Password:   hashedPassword,
			ProfilePic: profilePictureFilename,
		}

		userID, err := h.userRepo.CreateUser(newUser)
		if err != nil {
			http.Error(w, fmt.Sprintf("Errore nella registrazione: %v", err), http.StatusInternalServerError)
			return
		}

		// Crea una sessione e salva il cookie
		sessionID, _ := h.sm.CreateSession(userID)
		http.SetCookie(w, &http.Cookie{
			Name:  "session_id",
			Value: sessionID,
			Path:  "/", //il cookie è valido per tutto il dominio e tutti i percorsi del sito.
		})

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{  //encode prende un oggetto in go e lo converte in json
			"message":         "Registrazione completata con successo",
			"profile_picture": profilePictureFilename,
		})
	}
}

// LoginHandler gestisce il login degli utenti con controllo ban
func (h *AuthHandler) LoginHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var loginData LoginRequest
		err := json.NewDecoder(r.Body).Decode(&loginData) //decoder legge JSON e trasforma in dati Go
		if err != nil {
			h.respondWithError(w, "Dati non validi", http.StatusBadRequest)
			return
		}

		fmt.Printf("[LOGIN] Tentativo login per: %s\n", loginData.EmailOrUsername)

		// Verifica le credenziali dell'utente
		userID, err := h.userRepo.VerifyUser(loginData.EmailOrUsername, loginData.Password)
		if err != nil {
			fmt.Printf("[LOGIN] Credenziali errate per %s: %v\n", loginData.EmailOrUsername, err)
			h.respondWithError(w, "Credenziali errate", http.StatusUnauthorized)
			return
		}

		fmt.Printf("[LOGIN] Credenziali valide per userID: %d\n", userID)

		// Controllo ban
		isBanned, banInfo, err := h.banRepo.IsUserBanned(userID)
		if err != nil {
			fmt.Printf("[LOGIN] Errore controllo ban per userID %d: %v\n", userID, err)
			h.respondWithError(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		if isBanned && banInfo != nil {
			fmt.Printf("[LOGIN] Access denied - User %d is banned\n", userID)

			banResponse := &BanInfo{
				Reason:   banInfo.Reason,
				BannedAt: banInfo.BannedAt,
			}

			message := "Il tuo account è stato bannato permanentemente."

			response := LoginResponse{
				Success: false,
				Error:   message,
				BanInfo: banResponse,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Controllo attivo
		isActive, err := h.userRepo.IsUserActive(userID)
		if err != nil {
			fmt.Printf("[LOGIN] Error checking active status for userID %d: %v\n", userID, err)
			h.respondWithError(w, "Errore interno del server", http.StatusInternalServerError)
			return
		}

		if !isActive {
			fmt.Printf("[LOGIN] ❌ Access denied - User %d is not active\n", userID)
			h.respondWithError(w, "Account non attivo. Contatta l'amministratore.", http.StatusForbidden)
			return
		}

		// Crea sessione
		fmt.Printf("[LOGIN] Checks passed, creating session for userID: %d\n", userID)

		sessionID, err := h.sm.CreateSession(userID)
		if err != nil {
			fmt.Printf("[LOGIN] Error creating session for userID %d: %v\n", userID, err)
			h.respondWithError(w, "Errore nella creazione della sessione", http.StatusInternalServerError)
			return
		}

		// Imposta il cookie
		http.SetCookie(w, &http.Cookie{
			Name:  "session_id",
			Value: sessionID,
			Path:  "/",
		})

		fmt.Printf("[LOGIN] Login successfully completed for userID %d, SessionID: %s\n", userID, sessionID)

		response := LoginResponse{
			Success: true,
			Message: "Login riuscito",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// LogoutHandler invalida la sessione
func (h *AuthHandler) LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Sessione non trovata", http.StatusBadRequest)
			return
		}

		h.sm.DeleteSession(cookie.Value)

		// Elimina il cookie sul client
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

// ProfileBySessionHandler restituisce i dati del profilo come JSON
func (h *AuthHandler) ProfileBySessionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			log.Printf("ProfileBySessionHandler: session_id cookie not found or invalid")
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		user, err := h.userRepo.GetUserProfile(fmt.Sprintf("%d", userID))
		if err != nil {
			log.Printf("ProfileBySessionHandler: UserID=%d not found, err=%v\n", userID, err)
			http.Error(w, "Utente non trovato", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

// UserHandler restituisce solo l'email dell'utente autenticato
func (h *AuthHandler) UserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Session not found", http.StatusUnauthorized)
			return
		}

		user, err := h.userRepo.GetUserProfile(fmt.Sprintf("%d", userID))
		if err != nil {
			log.Printf("UserHandler: UserID=%d not found, err=%v\n", userID, err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		response := map[string]string{
			"email": user.Email,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetUserByEmailHandler trova un utente tramite email
func (h *AuthHandler) GetUserByEmailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		if email == "" {
			http.Error(w, "Email parameter is required", http.StatusBadRequest)
			return
		}

		user, err := h.userRepo.GetUserByEmail(email)
		switch {
		case err == sql.ErrNoRows:
			http.Error(w, "Utente non trovato", http.StatusNotFound)
		case err != nil:
			log.Printf("GetUserByEmailHandler: DB error for email %s: %v\n", email, err)
			http.Error(w, "Errore del database: "+err.Error(), http.StatusInternalServerError)
		default:
			log.Printf("GetUserByEmailHandler: Found user %s, isAdmin=%t\n", user.Email, user.IsAdmin)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
		}
	}
}

// UpdatePasswordHandler aggiorna la password dell'utente
func (h *AuthHandler) UpdatePasswordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Utente non autenticato", http.StatusUnauthorized)
			return
		}

		var req struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Formato richiesta non valido", http.StatusBadRequest)
			return
		}

		// Verifica la password corrente
		valid, err := h.userRepo.VerifyCurrentPassword(userID, req.CurrentPassword)
		if err != nil || !valid {
			http.Error(w, "Password corrente non valida", http.StatusUnauthorized)
			return
		}

		// Aggiorna la password
		if err := h.userRepo.UpdateUserPassword(userID, req.NewPassword); err != nil {
			http.Error(w, "Errore durante l'aggiornamento", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Password aggiornata con successo"})
	}
}

// ServeProfilePicture serve l'immagine del profilo
func (h *AuthHandler) ServeProfilePicture() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Path[len("/images/"):]
		filePath := filepath.Join("uploads/profile_pictures", filename)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.Error(w, "Immagine non trovata", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, filePath)
	}
}

// GetUserEmailHandler ottiene l'email dell'utente corrente
func (h *AuthHandler) GetUserEmailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := middleware.GetUserIDFromSession(r, h.sm)
		if err != nil {
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		user, err := h.userRepo.GetUserProfile(fmt.Sprintf("%d", userID))
		if err != nil {
			http.Error(w, "Utente non trovato", http.StatusNotFound)
			return
		}

		response := map[string]interface{}{
			"success":    true,
			"user_email": user.Email,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// Helper function per rispondere con errore
func (h *AuthHandler) respondWithError(w http.ResponseWriter, message string, statusCode int) {
	response := LoginResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}