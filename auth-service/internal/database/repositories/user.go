package repositories

import (
	"database/sql"
	"errors"
	"fmt"

	"trovagiocatoriAuth/internal/models"
	"trovagiocatoriAuth/internal/utils"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserisce un nuovo utente nel database
func (r *UserRepository) CreateUser(user models.User) (int64, error) {
	var userID int64
	err := r.db.QueryRow(`
		INSERT INTO users (nome, cognome, username, email, password, profile_picture)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		user.Nome, user.Cognome, user.Username, user.Email, user.Password, user.ProfilePic,
	).Scan(&userID)

	if err != nil {
		return 0, fmt.Errorf("errore nell'inserimento dell'utente: %v", err)
	}
	return userID, nil
}

// VerifyUser verifica le credenziali di login
func (r *UserRepository) VerifyUser(emailOrUsername, password string) (int64, error) {
	var userID int64
	var hashedPassword string

	err := r.db.QueryRow(`
		SELECT id, password FROM users WHERE email = $1 OR username = $2`,
		emailOrUsername, emailOrUsername).Scan(&userID, &hashedPassword)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("utente non trovato")
		}
		return 0, fmt.Errorf("errore durante la ricerca dell'utente: %v", err)
	}

	if !utils.CheckPasswordHash(password, hashedPassword) {
		return 0, errors.New("password errata")
	}

	return userID, nil
}

// GetUserProfile ottiene il profilo di un utente per ID
func (r *UserRepository) GetUserProfile(userID string) (models.User, error) {
	var user models.User
	var profilePic sql.NullString
	var isAdmin sql.NullBool
	
	query := `SELECT id, nome, cognome, username, email, password, profile_picture, COALESCE(is_admin, false) FROM users WHERE id = $1`
	err := r.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Nome, &user.Cognome, &user.Username, 
		&user.Email, &user.Password, &profilePic, &isAdmin)
	
	if err != nil {
		return user, err
	}
	
	if profilePic.Valid {
		user.ProfilePic = profilePic.String
	} else {
		user.ProfilePic = ""
	}
	
	if isAdmin.Valid {
		user.IsAdmin = isAdmin.Bool
	} else {
		user.IsAdmin = false
	}
	
	return user, nil
}

// GetUserByEmail trova un utente tramite email
func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	var profilePic sql.NullString
	
	err := r.db.QueryRow(`
        SELECT id, nome, cognome, username, email, profile_picture, COALESCE(is_admin, false) 
        FROM users 
        WHERE email = $1`, email).Scan(
		&user.ID, &user.Nome, &user.Cognome, &user.Username, &user.Email, &profilePic, &user.IsAdmin,
	)

	if profilePic.Valid {
		user.ProfilePic = profilePic.String
	} else {
		user.ProfilePic = ""
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserIDByEmail ottiene l'ID utente tramite email
func (r *UserRepository) GetUserIDByEmail(email string) (int64, error) {
	var userID int64
	err := r.db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

// CheckUserIsAdmin verifica se un utente è amministratore
func (r *UserRepository) CheckUserIsAdmin(userID int64) (bool, error) {
	var isAdmin bool
	err := r.db.QueryRow("SELECT COALESCE(is_admin, false) FROM users WHERE id = $1", userID).Scan(&isAdmin)
	if err != nil {
		return false, err
	}
	return isAdmin, nil
}

// IsUserActive verifica se un utente è attivo
func (r *UserRepository) IsUserActive(userID int64) (bool, error) {
	var isActive bool
	err := r.db.QueryRow("SELECT COALESCE(is_active, true) FROM users WHERE id = $1", userID).Scan(&isActive)
	if err != nil {
		return false, err
	}
	return isActive, nil
}

// VerifyCurrentPassword verifica la password corrente dell'utente
func (r *UserRepository) VerifyCurrentPassword(userID int64, currentPassword string) (bool, error) {
	var hashedPassword string
	err := r.db.QueryRow(
		"SELECT password FROM users WHERE id = $1",
		userID,
	).Scan(&hashedPassword)

	if err != nil {
		return false, err
	}

	return utils.CheckPasswordHash(currentPassword, hashedPassword), nil
}

// UpdateUserPassword aggiorna la password dell'utente
func (r *UserRepository) UpdateUserPassword(userID int64, newPassword string) error {
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		"UPDATE users SET password = $1 WHERE id = $2",
		hashedPassword,
		userID,
	)
	return err
}