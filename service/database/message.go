package database

import (
    "database/sql"
    "errors"
)

func (db *appdbimpl) CreateMessage(msg Message) error {
    _, err := db.c.Exec(`
        INSERT INTO messages (id, conversation_id, sender_id, created_at, content_type, text, photo_url, replied_to_message_id, status)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, msg.ID, msg.ConversationID, msg.SenderID, msg.CreatedAt, msg.ContentType, msg.Text, msg.PhotoURL, msg.RepliedToMessageID, msg.Status)
    return err
}

func (db *appdbimpl) GetMessageByID(id string) (*Message, error) {
    var m Message
    err := db.c.QueryRow(`
        SELECT id, conversation_id, sender_id, created_at, content_type, text, photo_url, replied_to_message_id, status
        FROM messages WHERE id = ?
    `, id).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.CreatedAt, &m.ContentType, &m.Text, &m.PhotoURL, &m.RepliedToMessageID, &m.Status)
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
        SELECT id, conversation_id, sender_id, created_at, content_type, text, photo_url, replied_to_message_id, status
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
        if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.CreatedAt, &m.ContentType, &m.Text, &m.PhotoURL, &m.RepliedToMessageID, &m.Status); err != nil {
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