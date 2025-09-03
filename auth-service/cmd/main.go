package main

import (
	"log"
	"net/http"
	"os"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/handlers"
	"trovagiocatoriAuth/internal/sessions"
)

func main() {

	// Leggi le variabili d'ambiente per il database
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	if host == "" || user == "" || password == "" || dbname == "" {
		log.Fatal("Database environment variables are not set")
	}

	// Inizializza il database
	database, err := db.InitPostgres(host, user, password, dbname)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}
	defer database.Conn.Close()

	// Inizializza il SessionManager
	sm := sessions.NewSessionManager()

	// Registrazione degli endpoint esistenti
	http.HandleFunc("/register", handlers.RegisterHandler(database, sm))
	http.HandleFunc("/login", handlers.LoginHandler(database, sm))
	http.HandleFunc("/logout", handlers.LogoutHandler(sm))

	http.HandleFunc("/profile", handlers.ProfileBySessionHandler(database, sm))
	http.HandleFunc("/images/", handlers.ServeProfilePicture)
	http.HandleFunc("/api/user", handlers.UserHandler(database, sm))

	http.HandleFunc("/api/user/by-email", handlers.GetUserByEmailHandler(database, sm))
	http.HandleFunc("/update-password", handlers.UpdatePasswordHandler(database, sm))

	// ENDPOINT PER I PREFERITI
	http.HandleFunc("/favorites/add", handlers.AddFavoriteHandler(database, sm))
	http.HandleFunc("/favorites/remove", handlers.RemoveFavoriteHandler(database, sm))
	http.HandleFunc("/favorites/check/", handlers.CheckFavoriteHandler(database, sm))
	http.HandleFunc("/favorites", handlers.GetUserFavoritesHandler(database, sm))

	//ENDPOINT PER LA PARTECIPAZIONE AGLI EVENTI
	http.HandleFunc("/events/join", handlers.JoinEventHandler(database, sm))
	http.HandleFunc("/events/leave", handlers.LeaveEventHandler(database, sm))
	http.HandleFunc("/events/check/", handlers.CheckParticipationHandler(database, sm))
	http.HandleFunc("/events/", handlers.GetEventParticipantsHandler(database, sm)) // events/{id}/participants

	// NUOVO: ENDPOINT PER OTTENERE LE PARTECIPAZIONI DELL'UTENTE (per il calendario)
	http.HandleFunc("/user/participations", handlers.GetUserParticipationsHandler(database, sm))

	log.Println("Auth service running on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
