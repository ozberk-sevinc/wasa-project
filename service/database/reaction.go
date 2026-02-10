package database

import (
	"database/sql"
	"errors"
)

func (db *appdbimpl) CreateReaction(r Reaction) error {
	// First, delete any existing reaction from this user on this message
	_, err := db.c.Exec(`
        DELETE FROM reactions WHERE message_id = ? AND user_id = ?
    `, r.MessageID, r.UserID)
	if err != nil {
		return err
	}

	// Then insert the new reaction
	_, err = db.c.Exec(`
        INSERT INTO reactions (id, message_id, user_id, emoji, created_at)
        VALUES (?, ?, ?, ?, ?)
    `, r.ID, r.MessageID, r.UserID, r.Emoji, r.CreatedAt)
	return err
}

func (db *appdbimpl) GetReactionByID(id string) (*Reaction, error) {
	var r Reaction
	err := db.c.QueryRow("SELECT id, message_id, user_id, emoji, created_at FROM reactions WHERE id = ?", id).Scan(&r.ID, &r.MessageID, &r.UserID, &r.Emoji, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (db *appdbimpl) GetReactionsByMessage(messageID string) ([]Reaction, error) {
	rows, err := db.c.Query("SELECT id, message_id, user_id, emoji, created_at FROM reactions WHERE message_id = ?", messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []Reaction
	for rows.Next() {
		var r Reaction
		if err := rows.Scan(&r.ID, &r.MessageID, &r.UserID, &r.Emoji, &r.CreatedAt); err != nil {
			return nil, err
		}
		reactions = append(reactions, r)
	}
	return reactions, rows.Err()
}

func (db *appdbimpl) GetUserReactionForMessage(messageID, userID string) (*Reaction, error) {
	var r Reaction
	err := db.c.QueryRow(`
		SELECT id, message_id, user_id, emoji, created_at 
		FROM reactions 
		WHERE message_id = ? AND user_id = ?
	`, messageID, userID).Scan(&r.ID, &r.MessageID, &r.UserID, &r.Emoji, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (db *appdbimpl) DeleteReaction(id string) error {
	_, err := db.c.Exec("DELETE FROM reactions WHERE id = ?", id)
	return err
}

// GetReactionsByConversation fetches all reactions for all messages in a conversation at once
func (db *appdbimpl) GetReactionsByConversation(conversationID string) ([]Reaction, error) {
	rows, err := db.c.Query(`
		SELECT r.id, r.message_id, r.user_id, r.emoji, r.created_at 
		FROM reactions r
		JOIN messages m ON r.message_id = m.id
		WHERE m.conversation_id = ?
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []Reaction
	for rows.Next() {
		var r Reaction
		if err := rows.Scan(&r.ID, &r.MessageID, &r.UserID, &r.Emoji, &r.CreatedAt); err != nil {
			return nil, err
		}
		reactions = append(reactions, r)
	}
	return reactions, rows.Err()
}
