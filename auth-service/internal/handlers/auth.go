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

	"golang.org/x/crypto/bcrypt"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/models"
	"trovagiocatoriAuth/internal/sessions"
)

// RegisterHandler gestisce la registrazione di un nuovo utente
func RegisterHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
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
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Errore durante l'hashing della password", http.StatusInternalServerError)
			return
		}

		// Gestione immagine profilo
		var profilePictureFilename string
		file, handler, err := r.FormFile("profile_picture")
		if err == nil { // Se è stato effettivamente caricato un file
			defer file.Close()

			uploadDir := "uploads/profile_pictures"
			// Crea la directory se non esiste
			if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
				if mkdirErr := os.MkdirAll(uploadDir, os.ModePerm); mkdirErr != nil {
					http.Error(w, fmt.Sprintf("Errore nella creazione della directory: %v", mkdirErr), http.StatusInternalServerError)
					return
				}
			}

			// Usa l'username per creare un nome univoco per il file
			extension := filepath.Ext(handler.Filename)
			profilePictureFilename = fmt.Sprintf("%s%s", username, extension)
			filePath := filepath.Join(uploadDir, profilePictureFilename)

			// Salva il file
			outFile, err := os.Create(filePath)
			if err != nil {
				http.Error(w, fmt.Sprintf("Errore nella creazione del file: %v", err), http.StatusInternalServerError)
				return
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, file)
			if err != nil {
				http.Error(w, fmt.Sprintf("Errore nella scrittura del file: %v", err), http.StatusInternalServerError)
				return
			}
			fmt.Printf("✔ Immagine salvata con successo: %s\n", filePath)
		}

		newUser := models.User{
			Nome:       nome,
			Cognome:    cognome,
			Username:   username,
			Email:      email,
			Password:   string(hashedPassword),
			ProfilePic: profilePictureFilename, // Salviamo solo il nome del file
		}

		userID, err := database.CreateUser(newUser)
		if err != nil {
			http.Error(w, fmt.Sprintf("Errore nella registrazione: %v", err), http.StatusInternalServerError)
			return
		}

		// Crea una sessione e salva il cookie
		sessionID, _ := sm.CreateSession(userID)
		http.SetCookie(w, &http.Cookie{
			Name:  "session_id",
			Value: sessionID,
			Path:  "/",
		})
		fmt.Printf("✔ SessionID: %s\n", sessionID)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"message":         "Registrazione completata con successo",
			"profile_picture": profilePictureFilename,
		})
	}
}

// ServeProfilePicture serve l'immagine del profilo dalla cartella uploads/profile_pictures
func ServeProfilePicture(w http.ResponseWriter, r *http.Request) {
	// Ottieni il nome del file dalla URL
	filename := r.URL.Path[len("/images/"):]

	// Costruisci il percorso completo
	filePath := filepath.Join("uploads/profile_pictures", filename)

	// Controlla se il file esiste
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Immagine non trovata", http.StatusNotFound)
		return
	}

	// Restituisce l'immagine al client
	http.ServeFile(w, r, filePath)
}

// ProfileBySessionHandler restituisce i dati del profilo come JSON,
// ricavando l'ID utente dal cookie di sessione senza richiedere "/profile/{id}"
func ProfileBySessionHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Recupera il cookie di sessione
		cookie, err := r.Cookie("session_id")
		if err != nil {
			log.Println("ProfileBySessionHandler: session_id cookie not found")
			http.Error(w, "Unauthorized: session_id non presente", http.StatusUnauthorized)
			return
		}

		log.Printf("ProfileBySessionHandler: Received session_id=%s\n", cookie.Value)

		// Ricava userID dalla sessione
		userID, err := sm.GetUserIDBySessionID(cookie.Value)
		if err != nil {
			log.Printf("ProfileBySessionHandler: Invalid session_id=%s\n", cookie.Value)
			http.Error(w, "Unauthorized: sessione non valida", http.StatusUnauthorized)
			return
		}

		log.Printf("ProfileBySessionHandler: Retrieved userID=%d for session_id=%s\n", userID, cookie.Value)

		// Recupera i dati utente dal database INCLUDENDO is_admin
		user, err := database.GetUserProfileWithAdmin(fmt.Sprintf("%d", userID))
		if err != nil {
			log.Printf("ProfileBySessionHandler: UserID=%d not found, err=%v\n", userID, err)
			http.Error(w, "Utente non trovato", http.StatusNotFound)
			return
		}

		log.Printf("ProfileBySessionHandler: Retrieved user data for userID=%d, isAdmin=%t\n", userID, user.IsAdmin)

		// Risponde in JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

// UserHandler restituisce solo l'email dell'utente autenticato
func UserHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Recupera il cookie di sessione
		cookie, err := r.Cookie("session_id")
		fmt.Printf("stampo il cookie che ho ricevuto: %+v\n", cookie)

		if err != nil {
			http.Error(w, "Session not found", http.StatusUnauthorized)
			return
		}

		// Ricava userID dalla sessione
		userID, err := sm.GetUserIDBySessionID(cookie.Value)
		//fmt.Printf("stampo user id che ho trovato: %+v\n", userID)

		//fmt.Printf("stampo errore che ho trovato: %+v\n", err)
		if err != nil {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		// Recupera il profilo dell'utente dal database USANDO LA FUNZIONE CON ADMIN
		user, err := database.GetUserProfileWithAdmin(fmt.Sprintf("%d", userID))
		if err != nil {
			log.Printf("UserHandler: UserID=%d not found, err=%v\n", userID, err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		//fmt.Printf("stampo user che ho trovato: %+v\n", user)

		// Risponde con l'email dell'utente
		response := map[string]string{
			"email": user.Email,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func GetUserByEmailHandler(database *db.Database, sm *sessions.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Estrazione email
		email := r.URL.Query().Get("email")
		if email == "" {
			http.Error(w, "Email parameter is required", http.StatusBadRequest)
			return
		}

		// Query al database con gestione NULL
		var user models.User
		var profilePic sql.NullString
		
		err := database.Conn.QueryRow(`
            SELECT id, nome, cognome, username, email, profile_picture, COALESCE(is_admin, false) 
            FROM users 
            WHERE email = $1`, email).Scan(
			&user.ID, &user.Nome, &user.Cognome, &user.Username, &user.Email, &profilePic, &user.IsAdmin,
		)

		// Gestisci il valore NULL per profile_picture
		if profilePic.Valid {
			user.ProfilePic = profilePic.String
		} else {
			user.ProfilePic = ""
		}

		switch {
		case err == sql.ErrNoRows:
			http.Error(w, "User not found", http.StatusNotFound)
		case err != nil:
			log.Printf("GetUserByEmailHandler: DB error for email %s: %v\n", email, err)
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		default:
			log.Printf("GetUserByEmailHandler: Found user %s, isAdmin=%t\n", user.Email, user.IsAdmin)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
		}
	}
}