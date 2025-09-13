package main

import (
	"log"
	"net/http"
	"os"

	"trovagiocatoriAuth/internal/db"
	"trovagiocatoriAuth/internal/handlers"
	"trovagiocatoriAuth/internal/services"
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

	// Crea le tabelle degli amici se non esistono
	if err := database.CreateFriendsTablesIfNotExists(); err != nil {
		log.Fatalf("Error creating friends tables: %v", err)
	}

	// Crea le tabelle degli inviti eventi se non esistono
	if err := database.CreateEventInvitesTableIfNotExists(); err != nil {
		log.Fatalf("Error creating event invites tables: %v", err)
	}

	// Crea le tabelle delle notifiche se non esistono
	if err := database.CreateNotificationsTableIfNotExists(); err != nil {
		log.Fatalf("Error creating notifications tables: %v", err)
	}

	// NUOVO: Crea le tabelle dei ban se non esistono
	if err := database.CreateBanTablesIfNotExists(); err != nil {
		log.Fatalf("Error creating ban tables: %v", err)
	}

	// NUOVO: Aggiorna tabella users con campi admin
	if err := database.CreateUsersTableWithAdminFields(); err != nil {
		log.Fatalf("Error updating users table with admin fields: %v", err)
	}

	// Inizializza e avvia il servizio di pulizia notifiche
	cleanupService := services.NewNotificationCleanupService(database)
	cleanupService.Start()
	defer cleanupService.Stop()

	// Stampa statistiche iniziali delle notifiche
	cleanupService.PrintStats()

	// Inizializza il SessionManager
	sm := sessions.NewSessionManager()

	// ========== ENDPOINT AUTENTICAZIONE ==========
	http.HandleFunc("/register", handlers.RegisterHandler(database, sm))
	http.HandleFunc("/login", handlers.LoginHandler(database, sm))
	http.HandleFunc("/logout", handlers.LogoutHandler(sm))

	// ========== ENDPOINT PROFILO UTENTE ==========
	http.HandleFunc("/profile", handlers.ProfileBySessionHandler(database, sm))
	http.HandleFunc("/images/", handlers.ServeProfilePicture)
	http.HandleFunc("/api/user", handlers.UserHandler(database, sm))
	http.HandleFunc("/api/user/by-email", handlers.GetUserByEmailHandler(database, sm))
	http.HandleFunc("/update-password", handlers.UpdatePasswordHandler(database, sm))

	// ========== ENDPOINT PREFERITI ==========
	http.HandleFunc("/favorites/add", handlers.AddFavoriteHandler(database, sm))
	http.HandleFunc("/favorites/remove", handlers.RemoveFavoriteHandler(database, sm))
	http.HandleFunc("/favorites/check/", handlers.CheckFavoriteHandler(database, sm))
	http.HandleFunc("/favorites", handlers.GetUserFavoritesHandler(database, sm))

	// ========== ENDPOINT PARTECIPAZIONE EVENTI ==========
	http.HandleFunc("/events/join", handlers.JoinEventHandler(database, sm))
	http.HandleFunc("/events/leave", handlers.LeaveEventHandler(database, sm))
	http.HandleFunc("/events/check/", handlers.CheckParticipationHandler(database, sm))
	http.HandleFunc("/events/", handlers.GetEventParticipantsHandler(database, sm)) // events/{id}/participants

	// ========== ENDPOINT PARTECIPAZIONI UTENTE ==========
	http.HandleFunc("/user/participations", handlers.GetUserParticipationsHandler(database, sm))
	http.HandleFunc("/user/email", handlers.GetUserEmailHandler(database, sm))

	// ========== ENDPOINT AMICI ==========
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

	// Endpoint aggiuntivi amici
	http.HandleFunc("/friends/search", handlers.SearchUsersHandler(database, sm))                  // GET
	http.HandleFunc("/friends/sent-requests", handlers.GetSentFriendRequestsHandler(database, sm)) // GET
	http.HandleFunc("/friends/cancel", handlers.CancelFriendRequestHandler(database, sm))          // POST

	// ========== ENDPOINT INVITI EVENTI ==========
	http.HandleFunc("/events/invite", handlers.SendEventInviteHandler(database, sm))          // POST
	http.HandleFunc("/events/invites", handlers.GetEventInvitesHandler(database, sm))         // GET
	http.HandleFunc("/events/invite/accept", handlers.AcceptEventInviteHandler(database, sm)) // POST
	http.HandleFunc("/events/invite/reject", handlers.RejectEventInviteHandler(database, sm)) // POST

	// Endpoint per amici disponibili per inviti
	http.HandleFunc("/friends/available-for-invite", handlers.GetAvailableFriendsForInviteHandler(database, sm)) // GET

	// ========== ENDPOINT NOTIFICHE ==========
	http.HandleFunc("/notifications", handlers.GetNotificationsHandler(database, sm))                    // GET - Lista notifiche
	http.HandleFunc("/notifications/summary", handlers.GetNotificationsSummaryHandler(database, sm))     // GET - Riassunto notifiche
	http.HandleFunc("/notifications/read", handlers.MarkNotificationAsReadHandler(database, sm))         // POST - Segna come letta
	http.HandleFunc("/notifications/read-all", handlers.MarkAllNotificationsAsReadHandler(database, sm)) // POST - Segna tutte come lette
	http.HandleFunc("/notifications/delete", handlers.DeleteNotificationHandler(database, sm))           // DELETE - Elimina notifica
	http.HandleFunc("/notifications/test", handlers.NotificationTestHandler(database, sm))               // POST - Test notifiche (solo dev)

	// ========== ENDPOINT AMMINISTRATORE ==========
	http.HandleFunc("/admin/posts/", handlers.AdminDeletePostHandler(database, sm))       // DELETE - Elimina post
	http.HandleFunc("/admin/comments/", handlers.AdminDeleteCommentHandler(database, sm)) // DELETE - Elimina commento
	http.HandleFunc("/admin/users", handlers.AdminGetUsersHandler(database, sm))          // GET - Lista utenti
	http.HandleFunc("/admin/users/", handlers.AdminToggleUserStatusHandler(database, sm)) // POST - Toggle status utente
	http.HandleFunc("/admin/stats", handlers.AdminStatsHandler(database, sm))             // GET - Statistiche dashboard

	// ========== NUOVI ENDPOINT BAN UTENTI ==========
	http.HandleFunc("/admin/bans", handlers.GetActiveBansHandler(database, sm))             // GET - Lista ban attivi
	http.HandleFunc("/admin/ban/user", handlers.BanUserHandler(database, sm))               // POST - Banna utente manualmente
	http.HandleFunc("/admin/unban/", handlers.UnbanUserHandler(database, sm))               // POST - Rimuovi ban (URL: /admin/unban/{userID})
	http.HandleFunc("/admin/ban/info/", handlers.GetUserBanHandler(database, sm))           // GET - Info ban utente
	http.HandleFunc("/admin/ban/history/", handlers.GetUserBanHistoryHandler(database, sm)) // GET - Cronologia ban utente
	http.HandleFunc("/admin/ban/stats", handlers.GetBanStatsHandler(database, sm))          // GET - Statistiche ban

	// ========== AVVIO SERVER ==========
	log.Println("🔔 Sistema notifiche attivato!")
	log.Println("🤝 Sistema amici configurato!")
	log.Println("🎉 Sistema inviti eventi configurato!")
	log.Println("🔧 Endpoint amministratore configurati!")
	log.Println("🚫 Sistema ban utenti attivato!") // NUOVO LOG
	log.Println("🚀 Auth service running on port 8080")

	// ========== AVVIO SERVER ==========
	log.Println("🔔 Sistema notifiche attivato!")
	log.Println("🤝 Sistema amici configurato!")
	log.Println("🎉 Sistema inviti eventi configurato!")
	log.Println("🔧 Endpoint amministratore configurati!")
	log.Println("🚀 Auth service running on port 8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
