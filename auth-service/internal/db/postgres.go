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
		profile_picture TEXT,
		is_admin BOOLEAN DEFAULT FALSE
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione del database: %v", err)
	}

	// Inserisci l'utente amministratore se non esiste

	_, err = db.Exec(`
	INSERT INTO users (nome, cognome, username, email, password, is_admin, profile_picture) 
	VALUES ('Admin', 'Sistema', 'admin', 'admin@trovagiocatori.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', TRUE, NULL)
	ON CONFLICT (email) DO UPDATE SET is_admin = TRUE;
	`)
	if err != nil {
		return fmt.Errorf("errore nell'inserimento dell'admin: %v", err)
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

	// Nuova tabella per le iscrizioni agli eventi
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS event_participants (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		post_id INTEGER NOT NULL,
		registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		status VARCHAR(20) DEFAULT 'confirmed',
		UNIQUE(user_id, post_id)
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella event_participants: %v", err)
	}

	fmt.Println("✔ Database migrato con successo!")
	fmt.Println("✔ Admin creato: admin@trovagiocatori.com / password")
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
	var user models.User
	query := `SELECT id, nome, cognome, username, email, password, profile_picture, COALESCE(is_admin, false) FROM users WHERE id = $1`
	err := db.Conn.QueryRow(query, userID).Scan(
		&user.ID, &user.Nome, &user.Cognome, &user.Username, 
		&user.Email, &user.Password, &user.ProfilePic, &user.IsAdmin)
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

// FUNZIONI PER I PREFERITI

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

// FUNZIONI PER LA PARTECIPAZIONE AGLI EVENTI

// JoinEvent - Iscrive un utente a un evento
func (db *Database) JoinEvent(userID int64, postID int) error {
	// Controlla se l'utente è già iscritto
	isAlreadyParticipant, err := db.IsEventParticipant(userID, postID)
	if err != nil {
		return err
	}
	
	if isAlreadyParticipant {
		return fmt.Errorf("utente già iscritto a questo evento")
	}

	_, err = db.Conn.Exec(`
		INSERT INTO event_participants (user_id, post_id, status) 
		VALUES ($1, $2, 'confirmed')`,
		userID, postID)
	return err
}

// LeaveEvent - Disiscrive un utente da un evento
func (db *Database) LeaveEvent(userID int64, postID int) error {
	_, err := db.Conn.Exec(`
		DELETE FROM event_participants 
		WHERE user_id = $1 AND post_id = $2`,
		userID, postID)
	return err
}

// IsEventParticipant - Controlla se un utente è iscritto a un evento
func (db *Database) IsEventParticipant(userID int64, postID int) (bool, error) {
	var count int
	err := db.Conn.QueryRow(`
		SELECT COUNT(*) FROM event_participants 
		WHERE user_id = $1 AND post_id = $2 AND status = 'confirmed'`,
		userID, postID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetEventParticipants - Ottiene tutti i partecipanti di un evento con le loro informazioni
func (db *Database) GetEventParticipants(postID int) ([]map[string]interface{}, error) {
	rows, err := db.Conn.Query(`
		SELECT 
			u.id, u.username, u.nome, u.cognome, u.email, u.profile_picture,
			ep.registered_at
		FROM event_participants ep
		JOIN users u ON ep.user_id = u.id
		WHERE ep.post_id = $1 AND ep.status = 'confirmed'
		ORDER BY ep.registered_at ASC`,
		postID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []map[string]interface{}
	for rows.Next() {
		var userID int64
		var username, nome, cognome, email, profilePic, registeredAt string
		
		err := rows.Scan(&userID, &username, &nome, &cognome, &email, &profilePic, &registeredAt)
		if err != nil {
			return nil, err
		}

		participant := map[string]interface{}{
			"user_id":         userID,
			"username":        username,
			"nome":            nome,
			"cognome":         cognome,
			"email":           email,
			"profile_picture": profilePic,
			"registered_at":   registeredAt,
		}
		participants = append(participants, participant)
	}

	return participants, rows.Err()
}

// GetEventParticipantCount - Ottiene il numero di partecipanti iscritti a un evento
func (db *Database) GetEventParticipantCount(postID int) (int, error) {
	var count int
	err := db.Conn.QueryRow(`
		SELECT COUNT(*) FROM event_participants 
		WHERE post_id = $1 AND status = 'confirmed'`,
		postID).Scan(&count)

	return count, err
}

// GetUserParticipations - Ottiene tutti gli eventi a cui un utente è iscritto
func (db *Database) GetUserParticipations(userID int64) ([]int, error) {
	rows, err := db.Conn.Query(`
		SELECT post_id FROM event_participants 
		WHERE user_id = $1 AND status = 'confirmed'
		ORDER BY registered_at DESC`,
		userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participations []int
	for rows.Next() {
		var postID int
		if err := rows.Scan(&postID); err != nil {
			return nil, err
		}
		participations = append(participations, postID)
	}

	return participations, rows.Err()
}

// CheckUserIsAdmin verifica se un utente è amministratore
func (db *Database) CheckUserIsAdmin(userID int64) (bool, error) {
	var isAdmin bool
	err := db.Conn.QueryRow("SELECT COALESCE(is_admin, false) FROM users WHERE id = $1", userID).Scan(&isAdmin)
	if err != nil {
		return false, err
	}
	return isAdmin, nil
}

// GetUserProfileWithAdmin ottiene il profilo utente incluso lo status admin
func (db *Database) GetUserProfileWithAdmin(userID string) (models.User, error) {
	var user models.User
	query := `SELECT id, nome, cognome, username, email, password, profile_picture, COALESCE(is_admin, false) FROM users WHERE id = $1`
	err := db.Conn.QueryRow(query, userID).Scan(
		&user.ID, &user.Nome, &user.Cognome, &user.Username, 
		&user.Email, &user.Password, &user.ProfilePic, &user.IsAdmin)
	if err != nil {
		return user, err
	}
	return user, nil
}