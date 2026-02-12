package database

import (
	"database/sql"
	"errors"
)

func (db *appdbimpl) CreateMessage(msg Message) error {
	_, err := db.c.Exec(`
        INSERT INTO messages (id, conversation_id, sender_id, created_at, content_type, text, photo_url, file_url, file_name, replied_to_message_id, status, is_forwarded)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, msg.ID, msg.ConversationID, msg.SenderID, msg.CreatedAt, msg.ContentType, msg.Text, msg.PhotoURL, msg.FileURL, msg.FileName, msg.RepliedToMessageID, msg.Status, msg.IsForwarded)
	return err
}

func (db *appdbimpl) GetMessageByID(id string) (*Message, error) {
	var m Message
	err := db.c.QueryRow(`
        SELECT id, conversation_id, sender_id, created_at, content_type, text, photo_url, file_url, file_name, replied_to_message_id, status, is_forwarded
        FROM messages WHERE id = ?
    `, id).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.CreatedAt, &m.ContentType, &m.Text, &m.PhotoURL, &m.FileURL, &m.FileName, &m.RepliedToMessageID, &m.Status, &m.IsForwarded)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (db *appdbimpl) GetMessagesByConversation(conversationID string) ([]Message, error) {
	rows, err := db.c.Query(`
        SELECT id, conversation_id, sender_id, created_at, content_type, text, photo_url, file_url, file_name, replied_to_message_id, status, is_forwarded
        FROM messages
        WHERE conversation_id = ?
        ORDER BY created_at ASC
    `, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.CreatedAt, &m.ContentType, &m.Text, &m.PhotoURL, &m.FileURL, &m.FileName, &m.RepliedToMessageID, &m.Status, &m.IsForwarded); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (db *appdbimpl) GetMessagesByConversationPaginated(conversationID string, limit, offset int) ([]Message, error) {
	rows, err := db.c.Query(`
        SELECT id, conversation_id, sender_id, created_at, content_type, text, photo_url, file_url, file_name, replied_to_message_id, status, is_forwarded
        FROM messages
        WHERE conversation_id = ?
        ORDER BY created_at ASC
        LIMIT ? OFFSET ?
    `, conversationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.CreatedAt, &m.ContentType, &m.Text, &m.PhotoURL, &m.FileURL, &m.FileName, &m.RepliedToMessageID, &m.Status, &m.IsForwarded); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (db *appdbimpl) DeleteMessage(id string) error {
	_, err := db.c.Exec("DELETE FROM messages WHERE id = ?", id)
	return err
}

func (db *appdbimpl) UpdateMessageStatus(id, status string) error {
	_, err := db.c.Exec("UPDATE messages SET status = ? WHERE id = ?", status, id)
	return err
}

// MarkMessagesAsReceived updates all messages NOT sent by userID to "received" status
// This is called when a user fetches their conversation list (one checkmark)
func (db *appdbimpl) MarkMessagesAsReceived(userID string) error {
	_, err := db.c.Exec(`
        UPDATE messages 
        SET status = 'received' 
        WHERE status = 'sent' 
        AND sender_id != ?
        AND conversation_id IN (
            SELECT conversation_id FROM conversation_participants WHERE user_id = ?
        )
    `, userID, userID)
	return err
}

// MarkMessagesAsRead updates all messages NOT sent by userID in a conversation to "read" status
// This is called when a user opens a specific conversation (two checkmarks)
func (db *appdbimpl) MarkMessagesAsRead(conversationID, userID string) error {
	// Get all messages in the conversation not sent by this user
	rows, err := db.c.Query(`
		SELECT id FROM messages 
		WHERE conversation_id = ? 
		AND sender_id != ?
		AND status IN ('sent', 'received')
	`, conversationID, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Mark each message as read by this user
	for rows.Next() {
		var messageID string
		if err := rows.Scan(&messageID); err != nil {
			return err
		}
		if err := db.MarkMessageReadByUser(messageID, userID); err != nil {
			return err
		}
	}
	return rows.Err()
}

// MarkMessageReadByUser records that a specific user has read a specific message
func (db *appdbimpl) MarkMessageReadByUser(messageID, userID string) error {
	_, err := db.c.Exec(`
		INSERT OR REPLACE INTO message_reads (message_id, user_id, read_at)
		VALUES (?, ?, datetime('now'))
	`, messageID, userID)
	return err
}

// GetMessageStatus determines the status of a message based on who has read it
// Returns "read" if all participants (except sender) have read it, "received" if some have, "sent" otherwise
func (db *appdbimpl) GetMessageStatus(messageID string) (string, error) {
	// Get the message to find conversation and sender
	var conversationID, senderID string
	err := db.c.QueryRow(`
		SELECT conversation_id, sender_id FROM messages WHERE id = ?
	`, messageID).Scan(&conversationID, &senderID)
	if err != nil {
		return StatusSent, err
	}

	// Get total number of participants (excluding sender)
	var totalParticipants int
	err = db.c.QueryRow(`
		SELECT COUNT(*) FROM conversation_participants 
		WHERE conversation_id = ? AND user_id != ?
	`, conversationID, senderID).Scan(&totalParticipants)
	if err != nil {
		return StatusSent, err
	}

	// If no other participants, status is just "sent"
	if totalParticipants == 0 {
		return StatusSent, nil
	}

	// Count how many participants have read the message
	var readCount int
	err = db.c.QueryRow(`
		SELECT COUNT(*) FROM message_reads 
		WHERE message_id = ?
	`, messageID).Scan(&readCount)
	if err != nil {
		return StatusSent, err
	}

	// Determine status based on read counts
	switch {
	case readCount == 0:
		return StatusSent, nil
	case readCount >= totalParticipants:
		return StatusRead, nil
	default:
		return StatusReceived, nil
	}
}
