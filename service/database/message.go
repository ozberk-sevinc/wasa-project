package database

import (
	"database/sql"
	"errors"
)

func (db *appdbimpl) CreateMessage(msg Message) error {
	_, err := db.c.Exec(`
        INSERT INTO messages (id, conversation_id, sender_id, created_at, content_type, text, photo_url, file_url, file_name, replied_to_message_id, status)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, msg.ID, msg.ConversationID, msg.SenderID, msg.CreatedAt, msg.ContentType, msg.Text, msg.PhotoURL, msg.FileURL, msg.FileName, msg.RepliedToMessageID, msg.Status)
	return err
}

func (db *appdbimpl) GetMessageByID(id string) (*Message, error) {
	var m Message
	err := db.c.QueryRow(`
        SELECT id, conversation_id, sender_id, created_at, content_type, text, photo_url, file_url, file_name, replied_to_message_id, status
        FROM messages WHERE id = ?
    `, id).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.CreatedAt, &m.ContentType, &m.Text, &m.PhotoURL, &m.FileURL, &m.FileName, &m.RepliedToMessageID, &m.Status)
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
        SELECT id, conversation_id, sender_id, created_at, content_type, text, photo_url, file_url, file_name, replied_to_message_id, status
        FROM messages
        WHERE conversation_id = ?
        ORDER BY created_at DESC
    `, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.CreatedAt, &m.ContentType, &m.Text, &m.PhotoURL, &m.FileURL, &m.FileName, &m.RepliedToMessageID, &m.Status); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (db *appdbimpl) GetMessagesByConversationPaginated(conversationID string, limit, offset int) ([]Message, error) {
	rows, err := db.c.Query(`
        SELECT id, conversation_id, sender_id, created_at, content_type, text, photo_url, file_url, file_name, replied_to_message_id, status
        FROM messages
        WHERE conversation_id = ?
        ORDER BY created_at DESC
        LIMIT ? OFFSET ?
    `, conversationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.CreatedAt, &m.ContentType, &m.Text, &m.PhotoURL, &m.FileURL, &m.FileName, &m.RepliedToMessageID, &m.Status); err != nil {
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
	_, err := db.c.Exec(`
        UPDATE messages 
        SET status = 'read' 
        WHERE conversation_id = ? 
        AND sender_id != ?
        AND status IN ('sent', 'received')
    `, conversationID, userID)
	return err
}
