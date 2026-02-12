package database

import (
	"database/sql"
	"errors"
)

func (db *appdbimpl) CreateConversation(id, convType, name string, createdBy *string, createdAt string) error {
	_, err := db.c.Exec("INSERT INTO conversations (id, type, name, created_by, created_at) VALUES (?, ?, ?, ?, ?)", id, convType, name, createdBy, createdAt)
	return err
}

func (db *appdbimpl) GetConversationByID(id string) (*Conversation, error) {
	var c Conversation
	err := db.c.QueryRow("SELECT id, type, name, photo_url, created_by, created_at FROM conversations WHERE id = ?", id).Scan(&c.ID, &c.Type, &c.Name, &c.PhotoURL, &c.CreatedBy, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *appdbimpl) GetConversationsByUser(userID string) ([]Conversation, error) {
	rows, err := db.c.Query(`
        SELECT c.id, c.type, c.name, c.photo_url, c.created_by, c.created_at
        FROM conversations c
        JOIN conversation_participants cp ON c.id = cp.conversation_id
        WHERE cp.user_id = ?
        ORDER BY (
            SELECT MAX(m.created_at) FROM messages m WHERE m.conversation_id = c.id
        ) DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &c.PhotoURL, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func (db *appdbimpl) AddParticipant(conversationID, userID string) error {
	_, err := db.c.Exec("INSERT OR IGNORE INTO conversation_participants (conversation_id, user_id) VALUES (?, ?)", conversationID, userID)
	return err
}

func (db *appdbimpl) RemoveParticipant(conversationID, userID string) error {
	_, err := db.c.Exec("DELETE FROM conversation_participants WHERE conversation_id = ? AND user_id = ?", conversationID, userID)
	return err
}

func (db *appdbimpl) GetParticipants(conversationID string) ([]User, error) {
	rows, err := db.c.Query(`
        SELECT u.id, u.name, u.display_name, u.photo_url
        FROM users u
        JOIN conversation_participants cp ON u.id = cp.user_id
        WHERE cp.conversation_id = ?
    `, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.DisplayName, &u.PhotoURL); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *appdbimpl) IsParticipant(conversationID, userID string) (bool, error) {
	var count int
	err := db.c.QueryRow("SELECT COUNT(*) FROM conversation_participants WHERE conversation_id = ? AND user_id = ?", conversationID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *appdbimpl) GetDirectConversation(userID1, userID2 string) (*Conversation, error) {
	var convID string
	err := db.c.QueryRow(`
        SELECT cp1.conversation_id
        FROM conversation_participants cp1
        JOIN conversation_participants cp2 ON cp1.conversation_id = cp2.conversation_id
        JOIN conversations c ON c.id = cp1.conversation_id
        WHERE cp1.user_id = ? AND cp2.user_id = ? AND c.type = 'direct'
    `, userID1, userID2).Scan(&convID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return db.GetConversationByID(convID)
}

func (db *appdbimpl) UpdateConversationName(conversationID, name string) error {
	_, err := db.c.Exec("UPDATE conversations SET name = ? WHERE id = ?", name, conversationID)
	return err
}

func (db *appdbimpl) UpdateConversationPhoto(conversationID string, photoURL *string) error {
	_, err := db.c.Exec("UPDATE conversations SET photo_url = ? WHERE id = ?", photoURL, conversationID)
	return err
}

// GetConversationSummariesByUser returns conversation summaries with last message info
func (db *appdbimpl) GetConversationSummariesByUser(userID string) ([]ConversationSummary, error) {
	rows, err := db.c.Query(`
        SELECT 
            c.id,
            c.type,
            c.name,
            c.photo_url,
            m.created_at,
            m.text,
            m.content_type
        FROM conversations c
        JOIN conversation_participants cp ON c.id = cp.conversation_id
        LEFT JOIN messages m ON m.id = (
            SELECT id FROM messages WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1
        )
        WHERE cp.user_id = ?
        ORDER BY m.created_at DESC NULLS LAST
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ConversationSummary
	for rows.Next() {
		var s ConversationSummary
		var contentType *string
		if err := rows.Scan(&s.ID, &s.Type, &s.Title, &s.PhotoURL, &s.LastMessageAt, &s.LastMessageSnippet, &contentType); err != nil {
			return nil, err
		}
		// Set LastMessageIsPhoto based on content type
		s.LastMessageIsPhoto = contentType != nil && *contentType == "photo"
		// If it's a photo, set snippet to "[photo]"
		if s.LastMessageIsPhoto {
			snippet := "[photo]"
			s.LastMessageSnippet = &snippet
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// GetLastMessage returns the most recent message in a conversation
func (db *appdbimpl) GetLastMessage(conversationID string) (*Message, error) {
	var m Message
	err := db.c.QueryRow(`
        SELECT id, conversation_id, sender_id, created_at, content_type, text, photo_url, replied_to_message_id, status
        FROM messages
        WHERE conversation_id = ?
        ORDER BY created_at DESC
        LIMIT 1
    `, conversationID).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.CreatedAt, &m.ContentType, &m.Text, &m.PhotoURL, &m.RepliedToMessageID, &m.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
