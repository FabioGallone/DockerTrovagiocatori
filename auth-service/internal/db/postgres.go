package db

import (
	"database/sql"
	"errors"
	"fmt"

	"trovagiocatoriAuth/internal/models"

	_ "github.com/lib/pq" // Driver PostgreSQL
	"golang.org/x/crypto/bcrypt"
)

type Database struct {
	Conn *sql.DB
}

func InitPostgres(host, user, password, dbname string) (*Database, error) {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", host, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Verifica la connessione
	if err := db.Ping(); err != nil {
		return nil, err
	}

	//  per creare la tabella se non esiste
	if err := migrateDatabase(db); err != nil {
		return nil, err
	}

	return &Database{Conn: db}, nil
}

func migrateDatabase(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		nome TEXT NOT NULL,
		cognome TEXT NOT NULL,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		profile_picture TEXT
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione del database: %v", err)
	}

	// Nuova tabella per i preferiti
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS user_favorites (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		post_id INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, post_id)
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella user_favorites: %v", err)
	}

	fmt.Println("✔ Database migrato con successo!")
	return nil
}

// CreateUser inserisce un nuovo utente nel database
func (db *Database) CreateUser(user models.User) (int64, error) {
	var userID int64
	err := db.Conn.QueryRow(`
		INSERT INTO users (nome, cognome, username, email, password, profile_picture)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		user.Nome, user.Cognome, user.Username, user.Email, user.Password, user.ProfilePic,
	).Scan(&userID)

	if err != nil {
		return 0, fmt.Errorf("errore nell'inserimento dell'utente: %v", err)
	}
	return userID, nil
}

func (db *Database) VerifyUser(emailOrUsername, password string) (int64, error) {
	var userID int64
	var hashedPassword string

	err := db.Conn.QueryRow(`
		SELECT id, password FROM users WHERE email = $1 OR username = $2`,
		emailOrUsername, emailOrUsername).Scan(&userID, &hashedPassword)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("utente non trovato")
		}
		return 0, fmt.Errorf("errore durante la ricerca dell'utente: %v", err)
	}

	// Confronta la password fornita dall'utente con l'hash salvato nel database
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return 0, errors.New("password errata")
	}

	return userID, nil
}

func (db *Database) GetUserProfile(userID string) (models.User, error) {
	print("sto entrando qua dentro")
	var user models.User
	query := `SELECT id, nome, cognome, username, email, password, profile_picture FROM users WHERE id = $1`
	err := db.Conn.QueryRow(query, userID).Scan(&user.ID, &user.Nome, &user.Cognome, &user.Username, &user.Email, &user.Password, &user.ProfilePic)
	if err != nil {
		return user, err
	}
	return user, nil
}

func (d *Database) VerifyCurrentPassword(userID int64, currentPassword string) (bool, error) {
	var hashedPassword string
	err := d.Conn.QueryRow(
		"SELECT password FROM users WHERE id = $1",
		userID,
	).Scan(&hashedPassword)

	if err != nil {
		return false, err
	}

	return checkPasswordHash(currentPassword, hashedPassword), nil
}

func (d *Database) UpdateUserPassword(userID int64, newPassword string) error {
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	_, err = d.Conn.Exec(
		"UPDATE users SET password = $1 WHERE id = $2",
		hashedPassword,
		userID,
	)
	return err
}

// NUOVE FUNZIONI PER I PREFERITI

// AddFavorite aggiunge un post ai preferiti dell'utente
func (db *Database) AddFavorite(userID int64, postID int) error {
	_, err := db.Conn.Exec(`
		INSERT INTO user_favorites (user_id, post_id) 
		VALUES ($1, $2) 
		ON CONFLICT (user_id, post_id) DO NOTHING`,
		userID, postID)
	return err
}

// RemoveFavorite rimuove un post dai preferiti dell'utente
func (db *Database) RemoveFavorite(userID int64, postID int) error {
	_, err := db.Conn.Exec(`
		DELETE FROM user_favorites 
		WHERE user_id = $1 AND post_id = $2`,
		userID, postID)
	return err
}

// IsFavorite controlla se un post è nei preferiti dell'utente
func (db *Database) IsFavorite(userID int64, postID int) (bool, error) {
	var count int
	err := db.Conn.QueryRow(`
		SELECT COUNT(*) FROM user_favorites 
		WHERE user_id = $1 AND post_id = $2`,
		userID, postID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetUserFavorites restituisce tutti i post ID preferiti di un utente
func (db *Database) GetUserFavorites(userID int64) ([]int, error) {
	rows, err := db.Conn.Query(`
		SELECT post_id FROM user_favorites 
		WHERE user_id = $1 
		ORDER BY created_at DESC`,
		userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favorites []int
	for rows.Next() {
		var postID int
		if err := rows.Scan(&postID); err != nil {
			return nil, err
		}
		favorites = append(favorites, postID)
	}

	return favorites, rows.Err()
}
