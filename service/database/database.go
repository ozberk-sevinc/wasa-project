/*
Package database is the middleware between the app database and the code. All data (de)serialization (save/load) from a
persistent database are handled here. Database specific logic should never escape this package.

To use this package you need to apply migrations to the database if needed/wanted, connect to it (using the database
data source name from config), and then initialize an instance of AppDatabase from the DB connection.

For example, this code adds a parameter in `webapi` executable for the database data source name (add it to the
main.WebAPIConfiguration structure):

	DB struct {
	    Filename string `conf:""`
	}

This is an example on how to migrate the DB and connect to it:

	// Start Database
	logger.Println("initializing database support")
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
	    logger.WithError(err).Error("error opening SQLite DB")
	    return fmt.Errorf("opening SQLite: %w", err)
	}
	defer func() {
	    logger.Debug("database stopping")
	    _ = db.Close()
	}()

Then you can initialize the AppDatabase and pass it to the api package.
*/
package database

import (
	"database/sql"
	"errors"
	"fmt"
)

// Message status constants
const (
	StatusSent     = "sent"
	StatusReceived = "received"
	StatusRead     = "read"
)

// User represents a WASAText user
type User struct {
	ID          string
	Name        string
	DisplayName *string
	PhotoURL    *string
}

// Conversation represents a conversation (direct or group)
type Conversation struct {
	ID        string
	Type      string // "direct" or "group"
	Name      string // group name or empty for direct
	PhotoURL  *string
	CreatedBy *string // User ID of group creator (null for direct conversations)
}

// Message represents a single message
type Message struct {
	ID                 string
	ConversationID     string
	SenderID           string
	CreatedAt          string
	ContentType        string // "text", "photo", "audio", "document", "file"
	Text               *string
	PhotoURL           *string
	FileURL            *string
	FileName           *string
	RepliedToMessageID *string
	Status             string // "sent", "received", "read"
	IsForwarded        bool
}

// Reaction represents an emoji reaction to a message
type Reaction struct {
	ID        string
	MessageID string
	UserID    string
	Emoji     string
	CreatedAt string
}

// ConversationSummary represents a conversation with last message info (for listing)
type ConversationSummary struct {
	ID                 string
	Type               string
	Title              string
	PhotoURL           *string
	LastMessageAt      *string
	LastMessageSnippet *string
	LastMessageIsPhoto bool
}

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	// User methods
	CreateUser(id, name string) error
	GetUserByID(id string) (*User, error)
	GetUserByName(name string) (*User, error)
	UpdateUsername(userID, newName string) error
	UpdateUserPhoto(userID string, photoURL *string) error
	SearchUsers(query string) ([]User, error)
	GetAllUsers() ([]User, error)
	GetUsersPaginated(limit, offset int) ([]User, error)
	GetUsersByIDs(ids []string) ([]User, error)

	// Conversation methods
	CreateConversation(id, convType, name string, createdBy *string) error
	GetConversationByID(id string) (*Conversation, error)
	GetConversationsByUser(userID string) ([]Conversation, error)
	GetConversationSummariesByUser(userID string) ([]ConversationSummary, error)
	GetLastMessage(conversationID string) (*Message, error)
	AddParticipant(conversationID, userID string) error
	RemoveParticipant(conversationID, userID string) error
	GetParticipants(conversationID string) ([]User, error)
	IsParticipant(conversationID, userID string) (bool, error)
	GetDirectConversation(userID1, userID2 string) (*Conversation, error)

	// Message methods
	CreateMessage(msg Message) error
	GetMessageByID(id string) (*Message, error)
	GetMessagesByConversation(conversationID string) ([]Message, error)
	GetMessagesByConversationPaginated(conversationID string, limit, offset int) ([]Message, error)
	DeleteMessage(id string) error
	UpdateMessageStatus(id, status string) error
	MarkMessagesAsReceived(userID string) error
	MarkMessagesAsRead(conversationID, userID string) error
	MarkMessageReadByUser(messageID, userID string) error
	GetMessageStatus(messageID string) (string, error)

	// Reaction methods
	CreateReaction(r Reaction) error
	GetReactionByID(id string) (*Reaction, error)
	GetReactionsByMessage(messageID string) ([]Reaction, error)
	GetReactionsByConversation(conversationID string) ([]Reaction, error)
	GetUserReactionForMessage(messageID, userID string) (*Reaction, error)
	DeleteReaction(id string) error

	// Group-specific methods
	UpdateConversationName(conversationID, name string) error
	UpdateConversationPhoto(conversationID string, photoURL *string) error

	Ping() error
}

type appdbimpl struct {
	c *sql.DB
}

// SQL schema definitions
const (
	createUsersTable = `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			display_name TEXT,
			photo_url TEXT
		)`

	createConversationsTable = `
		CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL CHECK (type IN ('direct', 'group')),
			name TEXT,
			photo_url TEXT,
			created_by TEXT,
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
		)`

	createConversationParticipantsTable = `
		CREATE TABLE IF NOT EXISTS conversation_participants (
			conversation_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			PRIMARY KEY (conversation_id, user_id),
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`

	createMessagesTable = `
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			content_type TEXT NOT NULL CHECK (content_type IN ('text', 'photo', 'audio', 'document', 'file')),
			text TEXT,
			photo_url TEXT,
			file_url TEXT,
			file_name TEXT,
			replied_to_message_id TEXT,
			status TEXT NOT NULL DEFAULT 'sent' CHECK (status IN ('sent', 'received', 'read')),
			is_forwarded INTEGER DEFAULT 0,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
			FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (replied_to_message_id) REFERENCES messages(id) ON DELETE SET NULL
		)`

	createReactionsTable = `
		CREATE TABLE IF NOT EXISTS reactions (
			id TEXT PRIMARY KEY,
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			emoji TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(message_id, user_id)
		)`

	createMessageReadsTable = `
		CREATE TABLE IF NOT EXISTS message_reads (
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			read_at TEXT NOT NULL,
			PRIMARY KEY (message_id, user_id),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`
)

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building a AppDatabase")
	}

	// Configure SQLite pragmas
	if err := configureSQLite(db); err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		return nil, err
	}

	return &appdbimpl{
		c: db,
	}, nil
}

// configureSQLite applies optimal SQLite settings
func configureSQLite(db *sql.DB) error {
	pragmas := map[string]string{
		"foreign_keys":     "ON",     // Enable foreign key constraints
		"journal_mode":     "WAL",    // Write-Ahead Logging for concurrent access
		"busy_timeout":     "30000",  // Wait 30 seconds on locked database
		"synchronous":      "NORMAL", // Balanced safety/performance
		"cache_size":       "-10000", // 10MB cache (negative = KB)
		"temp_store":       "memory", // Store temp tables in memory
		"mmap_size":        "0",      // Disable memory-mapped I/O
		"locking_mode":     "NORMAL", // Normal locking (not EXCLUSIVE)
		"read_uncommitted": "1",      // Allow reading uncommitted data for better concurrency
	}

	for key, value := range pragmas {
		query := fmt.Sprintf("PRAGMA %s = %s", key, value)
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("error executing %s: %w", query, err)
		}
	}

	return nil
}

// createTables creates all necessary database tables and indexes
func createTables(db *sql.DB) error {
	// Table definitions
	tables := []struct {
		name   string
		schema string
	}{
		{"users", createUsersTable},
		{"conversations", createConversationsTable},
		{"conversation_participants", createConversationParticipantsTable},
		{"messages", createMessagesTable},
		{"reactions", createReactionsTable},
		{"message_reads", createMessageReadsTable},
	}

	// Create tables
	for _, table := range tables {
		if _, err := db.Exec(table.schema); err != nil {
			return fmt.Errorf("error creating %s table: %w", table.name, err)
		}
	}

	// Create indexes
	indexes := []struct {
		name  string
		query string
	}{
		{"idx_messages_conversation", "CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id)"},
		{"idx_messages_created_at", "CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)"},
		{"idx_reactions_message", "CREATE INDEX IF NOT EXISTS idx_reactions_message ON reactions(message_id)"},
		{"idx_participants_user", "CREATE INDEX IF NOT EXISTS idx_participants_user ON conversation_participants(user_id)"},
		{"idx_users_name", "CREATE INDEX IF NOT EXISTS idx_users_name ON users(name)"},
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx.query); err != nil {
			return fmt.Errorf("error creating %s: %w", idx.name, err)
		}
	}

	return nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}
