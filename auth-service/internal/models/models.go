package models

import "time"

// User rappresenta un utente nel sistema
type User struct {
	ID         int64  `json:"id"`
	Nome       string `json:"nome"`
	Cognome    string `json:"cognome"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"-"`
	ProfilePic string `json:"profile_picture,omitempty"`
	IsAdmin    bool   `json:"is_admin"`
	IsActive   bool   `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AdminUserInfo rappresenta le informazioni di un utente per il pannello admin
type AdminUserInfo struct {
	ID                int64     `json:"id"`
	Username          string    `json:"username"`
	Nome              string    `json:"nome"`
	Cognome           string    `json:"cognome"`
	Email             string    `json:"email"`
	DataRegistrazione time.Time `json:"dataRegistrazione"`
	IsActive          bool      `json:"isActive"`
	IsAdmin           bool      `json:"isAdmin"`
	PostCreati        int       `json:"postCreati"`
	CommentiScritti   int       `json:"commentiScritti"`
}

// FriendInfo rappresenta le informazioni di un amico
type FriendInfo struct {
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	Nome         string `json:"nome"`
	Cognome      string `json:"cognome"`
	Email        string `json:"email"`
	ProfilePic   string `json:"profile_picture"`
	FriendsSince string `json:"friends_since"`
}

// FriendRequestInfo rappresenta una richiesta di amicizia
type FriendRequestInfo struct {
	RequestID   int64  `json:"request_id"`
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	Nome        string `json:"nome"`
	Cognome     string `json:"cognome"`
	Email       string `json:"email"`
	ProfilePic  string `json:"profile_picture"`
	RequestDate string `json:"request_date"`
	Status      string `json:"status"`
}

// UserSearchResult rappresenta un risultato di ricerca utenti
type UserSearchResult struct {
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
	Nome       string `json:"nome"`
	Cognome    string `json:"cognome"`
	Email      string `json:"email"`
	ProfilePic string `json:"profile_picture"`
}

// EventInviteInfo rappresenta un invito a un evento
type EventInviteInfo struct {
	InviteID             int64  `json:"invite_id"`
	PostID               int    `json:"post_id"`
	Message              string `json:"message"`
	CreatedAt            string `json:"created_at"`
	Status               string `json:"status"`
	SenderUsername       string `json:"sender_username"`
	SenderNome           string `json:"sender_nome"`
	SenderCognome        string `json:"sender_cognome"`
	SenderEmail          string `json:"sender_email"`
	SenderProfilePicture string `json:"sender_profile_picture"`
}

// NotificationType enum per i tipi di notifica
type NotificationType string

const (
	NotificationTypeFriendRequest NotificationType = "friend_request"
	NotificationTypeEventInvite   NotificationType = "event_invite"
	NotificationTypePostComment   NotificationType = "post_comment"
	NotificationTypeGeneral       NotificationType = "general"
)

// NotificationStatus enum per lo stato della notifica
type NotificationStatus string

const (
	NotificationStatusUnread NotificationStatus = "unread"
	NotificationStatusRead   NotificationStatus = "read"
)

// Notification rappresenta una notifica nel sistema
type Notification struct {
	ID         int64              `json:"id"`
	UserID     int64              `json:"user_id"`
	Type       NotificationType   `json:"type"`
	Title      string             `json:"title"`
	Message    string             `json:"message"`
	Status     NotificationStatus `json:"status"`
	RelatedID  *int64             `json:"related_id,omitempty"`
	SenderID   *int64             `json:"sender_id,omitempty"`
	SenderInfo *SenderInfo        `json:"sender_info,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	ExpiresAt  *time.Time         `json:"expires_at,omitempty"`
}

// SenderInfo contiene le informazioni del mittente della notifica
type SenderInfo struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	Nome        string `json:"nome"`
	Cognome     string `json:"cognome"`
	Email       string `json:"email"`
	ProfilePic  string `json:"profile_picture"`
	DisplayName string `json:"display_name"`
}

// NotificationSummary rappresenta un riassunto delle notifiche dell'utente
type NotificationSummary struct {
	UnreadCount      int  `json:"unread_count"`
	FriendRequests   int  `json:"friend_requests"`
	EventInvites     int  `json:"event_invites"`
	Comments         int  `json:"comments"`
	HasNotifications bool `json:"has_notifications"`
}

// UserBan rappresenta un ban di un utente
type UserBan struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	BannedByAdminID int64      `json:"banned_by_admin_id"`
	Reason          string     `json:"reason"`
	BannedAt        time.Time  `json:"banned_at"`
	UnbannedAt      *time.Time `json:"unbanned_at,omitempty"`
	UnbannedByID    *int64     `json:"unbanned_by_admin_id,omitempty"`
	IsActive        bool       `json:"is_active"`
	Notes           string     `json:"notes"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	AdminUsername   string     `json:"admin_username"`
}

// BanUserRequest rappresenta una richiesta di ban
type BanUserRequest struct {
    UserID int64  `json:"user_id"`
    Reason string `json:"reason,omitempty"` 
    Notes  string `json:"notes"`
}