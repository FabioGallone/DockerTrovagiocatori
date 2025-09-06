package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/handlers"
	"trovagiocatoriAuth/internal/sessions"
	"trovagiocatoriAuth/internal/services"
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

	// Crea le tabelle degli amici se non esistono
	if err := database.CreateFriendsTablesIfNotExists(); err != nil {
		log.Fatalf("Error creating friends tables: %v", err)
	}

	// Crea le tabelle degli inviti eventi se non esistono
	if err := database.CreateEventInvitesTableIfNotExists(); err != nil {
		log.Fatalf("Error creating event invites tables: %v", err)
	}

	// NUOVO: Crea le tabelle delle notifiche se non esistono
	if err := database.CreateNotificationsTableIfNotExists(); err != nil {
		log.Fatalf("Error creating notifications tables: %v", err)
	}

	// NUOVO: Inizializza e avvia il servizio di pulizia notifiche
	cleanupService := services.NewNotificationCleanupService(database)
	cleanupService.Start()
	defer cleanupService.Stop()

	// Stampa statistiche iniziali delle notifiche
	cleanupService.PrintStats()

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

	// ENDPOINT PER GLI AMICI (con notifiche integrate)
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

	// ENDPOINT AGGIUNTIVI AMICI
	http.HandleFunc("/friends/search", handlers.SearchUsersHandler(database, sm))                  // GET
	http.HandleFunc("/friends/sent-requests", handlers.GetSentFriendRequestsHandler(database, sm)) // GET
	http.HandleFunc("/friends/cancel", handlers.CancelFriendRequestHandler(database, sm))          // POST

	// ENDPOINT PER GLI INVITI EVENTI (con notifiche integrate)
	http.HandleFunc("/events/invite", handlers.SendEventInviteHandler(database, sm))          // POST
	http.HandleFunc("/events/invites", handlers.GetEventInvitesHandler(database, sm))         // GET
	http.HandleFunc("/events/invite/accept", handlers.AcceptEventInviteHandler(database, sm)) // POST
	http.HandleFunc("/events/invite/reject", handlers.RejectEventInviteHandler(database, sm)) // POST

	// NUOVI ENDPOINT PER LE NOTIFICHE
	http.HandleFunc("/notifications", handlers.GetNotificationsHandler(database, sm))                      // GET - Lista notifiche
	http.HandleFunc("/notifications/summary", handlers.GetNotificationsSummaryHandler(database, sm))       // GET - Riassunto notifiche
	http.HandleFunc("/notifications/read", handlers.MarkNotificationAsReadHandler(database, sm))           // POST - Segna come letta
	http.HandleFunc("/notifications/read-all", handlers.MarkAllNotificationsAsReadHandler(database, sm))   // POST - Segna tutte come lette
	http.HandleFunc("/notifications/delete", handlers.DeleteNotificationHandler(database, sm))             // DELETE - Elimina notifica
	http.HandleFunc("/notifications/test", handlers.NotificationTestHandler(database, sm))                 // POST - Test notifiche (solo dev)

	log.Println("🔔 Sistema notifiche attivato!")
	log.Println("Auth service running on port 8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}