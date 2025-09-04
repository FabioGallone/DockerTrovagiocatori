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

	// NUOVO: Crea le tabelle degli amici se non esistono
	if err := database.CreateFriendsTablesIfNotExists(); err != nil {
		log.Fatalf("Error creating friends tables: %v", err)
	}

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

	// ENDPOINT PER LA PARTECIPAZIONE AGLI EVENTI
	http.HandleFunc("/events/join", handlers.JoinEventHandler(database, sm))
	http.HandleFunc("/events/leave", handlers.LeaveEventHandler(database, sm))
	http.HandleFunc("/events/check/", handlers.CheckParticipationHandler(database, sm))
	http.HandleFunc("/events/", handlers.GetEventParticipantsHandler(database, sm)) // events/{id}/participants

	// ENDPOINT PER LE PARTECIPAZIONI DELL'UTENTE (per il calendario)
	http.HandleFunc("/user/participations", handlers.GetUserParticipationsHandler(database, sm))

	// ENDPOINT PER OTTENERE L'EMAIL DELL'UTENTE (per "I Miei Post")
	http.HandleFunc("/user/email", handlers.GetUserEmailHandler(database, sm))

	// NUOVI ENDPOINT PER GLI AMICI
	// Gestione richieste di amicizia
	http.HandleFunc("/friends/request", handlers.SendFriendRequestHandler(database, sm))  // POST
	http.HandleFunc("/friends/accept", handlers.AcceptFriendRequestHandler(database, sm)) // POST
	http.HandleFunc("/friends/reject", handlers.RejectFriendRequestHandler(database, sm)) // POST
	http.HandleFunc("/friends/remove", handlers.RemoveFriendHandler(database, sm))        // DELETE

	// Controllo stato amicizia
	http.HandleFunc("/friends/check", handlers.CheckFriendshipHandler(database, sm)) // GET

	// Liste e richieste
	http.HandleFunc("/friends/list", handlers.GetFriendsListHandler(database, sm))        // GET
	http.HandleFunc("/friends/requests", handlers.GetFriendRequestsHandler(database, sm)) // GET

	// ENDPOINT AGGIUNTIVI (opzionali)
	// http.HandleFunc("/friends/search", handlers.SearchUsersHandler(database, sm))               // GET
	// http.HandleFunc("/friends/sent-requests", handlers.GetSentFriendRequestsHandler(database, sm)) // GET
	// http.HandleFunc("/friends/cancel", handlers.CancelFriendRequestHandler(database, sm))       // POST
	// http.HandleFunc("/friends/mutual", handlers.GetMutualFriendsHandler(database, sm))          // GET

	log.Println("Auth service running on port 8080")
	log.Println("✅ Endpoints degli amici configurati:")
	log.Println("   POST /friends/request    - Invia richiesta amicizia")
	log.Println("   POST /friends/accept     - Accetta richiesta amicizia")
	log.Println("   POST /friends/reject     - Rifiuta richiesta amicizia")
	log.Println("   DELETE /friends/remove   - Rimuove amicizia")
	log.Println("   GET /friends/check       - Controlla se sono amici")
	log.Println("   GET /friends/list        - Lista amici")
	log.Println("   GET /friends/requests    - Richieste ricevute")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
