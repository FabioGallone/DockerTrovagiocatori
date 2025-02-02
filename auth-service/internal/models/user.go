package models

// User rappresenta un utente nel sistema
type User struct {
	ID         int64  `json:"id"`
	Nome       string `json:"nome"`
	Cognome    string `json:"cognome"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"-"`                         // Nascondiamo la password nella risposta JSON
	ProfilePic string `json:"profile_picture,omitempty"` // Percorso dell'immagine profilo (opzionale)
}
