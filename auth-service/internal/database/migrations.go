package database

import (
	"fmt"
	"log"
)

func (db *Database) runMigrations() error {
	// Esegui tutte le migrazioni in ordine
	migrations := []func() error{
		db.createUsersTable,
		db.createFavoritesTable,
		db.createEventParticipantsTable,
		db.createFriendsTablesIfNotExists,
		db.createEventInvitesTableIfNotExists,
		db.createNotificationsTableIfNotExists,
		db.createBanTablesIfNotExists,
		db.updateUsersTableWithAdminFields,
	}

	for i, migration := range migrations {
		if err := migration(); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	fmt.Println("All database migrations completed successfully")
	return nil
}

func (db *Database) createUsersTable() error {
	_, err := db.Conn.Exec(`
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
	_, err = db.Conn.Exec(`
	INSERT INTO users (nome, cognome, username, email, password, is_admin, profile_picture) 
	VALUES ('Admin', 'Sistema', 'admin', 'admin@trovagiocatori.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', TRUE, NULL)
	ON CONFLICT (email) DO UPDATE SET is_admin = TRUE;
	`)
	if err != nil {
		return fmt.Errorf("Errore nell'inserimento dell'admin: %v", err)
	}

	log.Println("Users table created and admin user initialized")
	return nil
}

func (db *Database) createFavoritesTable() error {
	_, err := db.Conn.Exec(`
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

	log.Println("✔ User favorites table created")
	return nil
}

func (db *Database) createEventParticipantsTable() error {
	_, err := db.Conn.Exec(`
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

	log.Println("✔ Event participants table created")
	return nil
}

func (db *Database) createFriendsTablesIfNotExists() error {
	// Tabella delle amicizie
	_, err := db.Conn.Exec(`
	CREATE TABLE IF NOT EXISTS friendships (
		id SERIAL PRIMARY KEY,
		user1_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		user2_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user1_id, user2_id),
		CHECK(user1_id != user2_id),
		CHECK(user1_id < user2_id)
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella friendships: %v", err)
	}

	// Tabella delle richieste di amicizia
	_, err = db.Conn.Exec(`
	CREATE TABLE IF NOT EXISTS friend_requests (
		id SERIAL PRIMARY KEY,
		sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		receiver_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(sender_id, receiver_id),
		CHECK(sender_id != receiver_id),
		CHECK(status IN ('pending', 'accepted', 'rejected', 'cancelled'))
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella friend_requests: %v", err)
	}

	// Indici per migliorare le performance
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_friendships_user1 ON friendships(user1_id)",
		"CREATE INDEX IF NOT EXISTS idx_friendships_user2 ON friendships(user2_id)",
		"CREATE INDEX IF NOT EXISTS idx_friend_requests_sender ON friend_requests(sender_id)",
		"CREATE INDEX IF NOT EXISTS idx_friend_requests_receiver ON friend_requests(receiver_id)",
		"CREATE INDEX IF NOT EXISTS idx_friend_requests_status ON friend_requests(status)",
	}

	for _, query := range indexQueries {
		_, err = db.Conn.Exec(query)
		if err != nil {
			return fmt.Errorf("errore nella creazione degli indici: %v", err)
		}
	}

	log.Println("✔ Friends tables created successfully")
	return nil
}

func (db *Database) createEventInvitesTableIfNotExists() error {
	_, err := db.Conn.Exec(`
	CREATE TABLE IF NOT EXISTS event_invites (
		id SERIAL PRIMARY KEY,
		sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		receiver_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		post_id INTEGER NOT NULL,
		message TEXT,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(receiver_id, post_id),
		CHECK(sender_id != receiver_id),
		CHECK(status IN ('pending', 'accepted', 'rejected'))
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella event_invites: %v", err)
	}

	// Indici per migliorare le performance
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_event_invites_sender ON event_invites(sender_id)",
		"CREATE INDEX IF NOT EXISTS idx_event_invites_receiver ON event_invites(receiver_id)",
		"CREATE INDEX IF NOT EXISTS idx_event_invites_post ON event_invites(post_id)",
		"CREATE INDEX IF NOT EXISTS idx_event_invites_status ON event_invites(status)",
	}

	for _, query := range indexQueries {
		_, err = db.Conn.Exec(query)
		if err != nil {
			return fmt.Errorf("errore nella creazione degli indici event_invites: %v", err)
		}
	}

	log.Println("✔ Event invites table created successfully")
	return nil
}

func (db *Database) createNotificationsTableIfNotExists() error {
	_, err := db.Conn.Exec(`
	CREATE TABLE IF NOT EXISTS notifications (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		type VARCHAR(50) NOT NULL,
		title VARCHAR(255) NOT NULL,
		message TEXT NOT NULL,
		status VARCHAR(20) DEFAULT 'unread',
		related_id INTEGER,
		sender_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP,
		CHECK(status IN ('unread', 'read')),
		CHECK(type IN ('friend_request', 'event_invite', 'post_comment', 'general'))
	);
	`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella notifications: %v", err)
	}

	// Indici per migliorare le performance
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_user_status ON notifications(user_id, status)",
	}

	for _, query := range indexQueries {
		_, err = db.Conn.Exec(query)
		if err != nil {
			return fmt.Errorf("errore nella creazione degli indici notifications: %v", err)
		}
	}

	log.Println("✔ Notifications table created successfully")
	return nil
}

func (db *Database) createBanTablesIfNotExists() error {
	// Tabella user_bans
	_, err := db.Conn.Exec(`
		CREATE TABLE IF NOT EXISTS user_bans (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			banned_by_admin_id INTEGER NOT NULL REFERENCES users(id) ON DELETE SET NULL,
			reason TEXT DEFAULT 'Ban amministrativo',
			banned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			unbanned_at TIMESTAMP NULL,
			unbanned_by_admin_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			is_active BOOLEAN DEFAULT TRUE,
			notes TEXT
		)`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella user_bans: %v", err)
	}

	// Tabella ban_history
	_, err = db.Conn.Exec(`
		CREATE TABLE IF NOT EXISTS ban_history (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			admin_id INTEGER NOT NULL REFERENCES users(id) ON DELETE SET NULL,
			action VARCHAR(20) NOT NULL,
			reason TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			ban_id INTEGER REFERENCES user_bans(id) ON DELETE SET NULL
		)`)
	if err != nil {
		return fmt.Errorf("errore nella creazione della tabella ban_history: %v", err)
	}

	// Indici per performance
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_user_bans_user_id ON user_bans(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_user_bans_active ON user_bans(is_active)",
		"CREATE INDEX IF NOT EXISTS idx_ban_history_user_id ON ban_history(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_ban_history_created_at ON ban_history(created_at)",
	}

	for _, query := range indexQueries {
		db.Conn.Exec(query)
	}

	log.Println("✔ Ban tables created successfully")
	return nil
}

func (db *Database) updateUsersTableWithAdminFields() error {
	// Aggiungi colonne se non esistono
	alterQueries := []string{
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
	}

	for _, query := range alterQueries {
		_, err := db.Conn.Exec(query)
		if err != nil {
			return fmt.Errorf("errore nell'aggiornamento tabella users: %v", err)
		}
	}

	// Aggiorna tutti gli utenti esistenti che non hanno created_at
	_, err := db.Conn.Exec(`
		UPDATE users 
		SET created_at = CURRENT_TIMESTAMP 
		WHERE created_at IS NULL
	`)
	if err != nil {
		return fmt.Errorf("errore nell'aggiornamento created_at: %v", err)
	}

	log.Println("✔ Users table updated with admin fields")
	return nil
}