package models

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
}