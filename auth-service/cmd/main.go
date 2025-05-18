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

	// Registrazione degli endpoint
	http.HandleFunc("/register", handlers.RegisterHandler(database, sm))
	http.HandleFunc("/login", handlers.LoginHandler(database, sm))
	http.HandleFunc("/logout", handlers.LogoutHandler(sm))

	http.HandleFunc("/profile", handlers.ProfileBySessionHandler(database, sm))
	http.HandleFunc("/images/", handlers.ServeProfilePicture)
	http.HandleFunc("/api/user", handlers.UserHandler(database, sm))

	http.HandleFunc("/api/user/by-email", handlers.GetUserByEmailHandler(database, sm))

	log.Println("Auth service running on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
