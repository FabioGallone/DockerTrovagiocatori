package main

import (
	"log"
	"net/http"

	"trovagiocatoriAuth/internal/config"
	"trovagiocatoriAuth/internal/database"
	"trovagiocatoriAuth/internal/database/repositories"
	"trovagiocatoriAuth/internal/handlers"
	"trovagiocatoriAuth/internal/middleware"
	"trovagiocatoriAuth/internal/services"
	"trovagiocatoriAuth/internal/sessions"
)

func main() {
	// Carica configurazione
	cfg := config.LoadConfig()

	// Inizializza il database
	db, err := database.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}
	defer db.Close()

	// Inizializza i repository
	userRepo := repositories.NewUserRepository(db.Conn)
	adminRepo := repositories.NewAdminRepository(db.Conn)
	friendRepo := repositories.NewFriendRepository(db.Conn)
	eventRepo := repositories.NewEventRepository(db.Conn)
	notificationRepo := repositories.NewNotificationRepository(db.Conn)
	banRepo := repositories.NewBanRepository(db.Conn)

	// Inizializza il SessionManager
	sm := sessions.NewSessionManager()

	// Inizializza i servizi
	cleanupService := services.NewNotificationCleanupService(notificationRepo)
	cleanupService.Start()
	defer cleanupService.Stop()


	// Inizializza gli handlers
	authHandler := handlers.NewAuthHandler(userRepo, banRepo, sm)
	friendHandler := handlers.NewFriendHandler(friendRepo, userRepo, notificationRepo, sm)
	eventHandler := handlers.NewEventHandler(eventRepo, userRepo, sm)
	notificationHandler := handlers.NewNotificationHandler(notificationRepo, sm)
	adminHandler := handlers.NewAdminHandler(adminRepo, userRepo, banRepo, sm)
	banHandler := handlers.NewBanHandler(banRepo, userRepo, sm)

	// Setup routes
	setupRoutes(authHandler, friendHandler, eventHandler, notificationHandler, adminHandler, banHandler, userRepo, sm)

	// Stampa messaggi di avvio
	log.Println("Notification system activated!")
	log.Println("Friends system configured!")
	log.Println("Event invitations system configured!")
	log.Println("Admin endpoint configured!")
	log.Println("User ban system activated!")
	log.Printf("Auth service running on port %s", cfg.Server.Port)

	// Avvia il server
	if err := http.ListenAndServe(":"+cfg.Server.Port, nil); err != nil {
		log.Fatal(err)
	}
}

func setupRoutes(
	authHandler *handlers.AuthHandler,
	friendHandler *handlers.FriendHandler,
	eventHandler *handlers.EventHandler,
	notificationHandler *handlers.NotificationHandler,
	adminHandler *handlers.AdminHandler,
	banHandler *handlers.BanHandler,
	userRepo *repositories.UserRepository,
	sm *sessions.SessionManager,
) {
	// ========== ENDPOINT AUTENTICAZIONE ==========
	http.HandleFunc("/register", authHandler.RegisterHandler())
	http.HandleFunc("/login", authHandler.LoginHandler())
	http.HandleFunc("/logout", authHandler.LogoutHandler())

	// ========== ENDPOINT PROFILO UTENTE ==========
	http.HandleFunc("/profile", authHandler.ProfileBySessionHandler())
	http.HandleFunc("/images/", authHandler.ServeProfilePicture())
	http.HandleFunc("/api/user", authHandler.UserHandler())
	http.HandleFunc("/api/user/by-email", authHandler.GetUserByEmailHandler())
	http.HandleFunc("/update-password", authHandler.UpdatePasswordHandler())

	// ========== ENDPOINT PREFERITI ==========
	http.HandleFunc("/favorites/add", eventHandler.AddFavoriteHandler())
	http.HandleFunc("/favorites/remove", eventHandler.RemoveFavoriteHandler())
	http.HandleFunc("/favorites/check/", eventHandler.CheckFavoriteHandler())
	http.HandleFunc("/favorites", eventHandler.GetUserFavoritesHandler())

	// ========== ENDPOINT PARTECIPAZIONE EVENTI ==========
	http.HandleFunc("/events/join", eventHandler.JoinEventHandler())
	http.HandleFunc("/events/leave", eventHandler.LeaveEventHandler())
	http.HandleFunc("/events/check/", eventHandler.CheckParticipationHandler())
	http.HandleFunc("/events/", eventHandler.GetEventParticipantsHandler())

	// ========== ENDPOINT PARTECIPAZIONI UTENTE ==========
	http.HandleFunc("/user/participations", eventHandler.GetUserParticipationsHandler())
	http.HandleFunc("/user/email", authHandler.GetUserEmailHandler())

	// ========== ENDPOINT AMICI ==========
	http.HandleFunc("/friends/request", friendHandler.SendFriendRequestHandler())
	http.HandleFunc("/friends/accept", friendHandler.AcceptFriendRequestHandler())
	http.HandleFunc("/friends/reject", friendHandler.RejectFriendRequestHandler())
	http.HandleFunc("/friends/remove", friendHandler.RemoveFriendHandler())
	http.HandleFunc("/friends/check", friendHandler.CheckFriendshipHandler())
	http.HandleFunc("/friends/list", friendHandler.GetFriendsListHandler())
	http.HandleFunc("/friends/requests", friendHandler.GetFriendRequestsHandler())
	http.HandleFunc("/friends/search", friendHandler.SearchUsersHandler())
	http.HandleFunc("/friends/sent-requests", friendHandler.GetSentFriendRequestsHandler())
	http.HandleFunc("/friends/cancel", friendHandler.CancelFriendRequestHandler())

	// ========== ENDPOINT INVITI EVENTI ==========
	http.HandleFunc("/events/invite", eventHandler.SendEventInviteHandler())
	http.HandleFunc("/events/invites", eventHandler.GetEventInvitesHandler())
	http.HandleFunc("/events/invite/accept", eventHandler.AcceptEventInviteHandler())
	http.HandleFunc("/events/invite/reject", eventHandler.RejectEventInviteHandler())
	http.HandleFunc("/friends/available-for-invite", eventHandler.GetAvailableFriendsForInviteHandler())

	// ========== ENDPOINT NOTIFICHE ==========
	http.HandleFunc("/notifications", notificationHandler.GetNotificationsHandler())
	http.HandleFunc("/notifications/summary", notificationHandler.GetNotificationsSummaryHandler())
	http.HandleFunc("/notifications/read", notificationHandler.MarkNotificationAsReadHandler())
	http.HandleFunc("/notifications/read-all", notificationHandler.MarkAllNotificationsAsReadHandler())
	http.HandleFunc("/notifications/delete", notificationHandler.DeleteNotificationHandler())
	http.HandleFunc("/notifications/test", notificationHandler.NotificationTestHandler())

	// ========== ENDPOINT AMMINISTRATORE ==========
	http.HandleFunc("/admin/posts/", middleware.RequireAdmin(userRepo, sm)(adminHandler.AdminDeletePostHandler()))
	http.HandleFunc("/admin/comments/", middleware.RequireAdmin(userRepo, sm)(adminHandler.AdminDeleteCommentHandler()))
	http.HandleFunc("/admin/users", middleware.RequireAdmin(userRepo, sm)(adminHandler.AdminGetUsersHandler()))
	http.HandleFunc("/admin/users/", middleware.RequireAdmin(userRepo, sm)(adminHandler.AdminToggleUserStatusHandler()))
	http.HandleFunc("/admin/stats", middleware.RequireAdmin(userRepo, sm)(adminHandler.AdminStatsHandler()))

	// ========== ENDPOINT BAN UTENTI ==========
	http.HandleFunc("/admin/bans", middleware.RequireAdmin(userRepo, sm)(banHandler.GetActiveBansHandler()))
	http.HandleFunc("/admin/ban/user", middleware.RequireAdmin(userRepo, sm)(banHandler.BanUserHandler()))
	http.HandleFunc("/admin/unban/", middleware.RequireAdmin(userRepo, sm)(banHandler.UnbanUserHandler()))
	http.HandleFunc("/admin/ban/info/", middleware.RequireAdmin(userRepo, sm)(banHandler.GetUserBanHandler()))
	http.HandleFunc("/admin/ban/history/", middleware.RequireAdmin(userRepo, sm)(banHandler.GetUserBanHistoryHandler()))
	http.HandleFunc("/admin/ban/stats", middleware.RequireAdmin(userRepo, sm)(banHandler.GetBanStatsHandler()))
}