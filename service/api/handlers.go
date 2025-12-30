package api

import (
	"encoding/json"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/ozberk-sevinc/wasa-project/service/api/reqcontext"
	"github.com/ozberk-sevinc/wasa-project/service/database"
	"github.com/ozberk-sevinc/wasa-project/service/globaltime"
)

// ============================================================================
// RESPONSE TYPES (matching api.yaml schemas)
// ============================================================================

// UserResponse matches the User schema
type UserResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	DisplayName *string `json:"displayName,omitempty"`
	PhotoURL    *string `json:"photoUrl,omitempty"`
}

// LoginRequest is the request body for POST /session
type LoginRequest struct {
	Name string `json:"name"`
}

// LoginResponse is the response for POST /session
type LoginResponse struct {
	Identifier string `json:"identifier"`
}

// SetUsernameRequest is the request body for PUT /me/username
type SetUsernameRequest struct {
	Name string `json:"name"`
}

// SearchUsersResponse is the response for GET /users
type SearchUsersResponse struct {
	Users []UserResponse `json:"users"`
}

// CreateConversationRequest is the request body for POST /conversations
type CreateConversationRequest struct {
	UserID string `json:"userId"`
}

// ============================================================================
// CONVERSATION RESPONSE TYPES
// ============================================================================

// ConversationSummaryResponse matches the ConversationSummary schema
type ConversationSummaryResponse struct {
	ID                 string  `json:"id"`
	Type               string  `json:"type"`
	Title              string  `json:"title"`
	PhotoURL           *string `json:"photoUrl,omitempty"`
	LastMessageAt      *string `json:"lastMessageAt,omitempty"`
	LastMessageSnippet *string `json:"lastMessageSnippet,omitempty"`
	LastMessageIsPhoto bool    `json:"lastMessageIsPhoto"`
}

// ReactionResponse matches the Reaction schema
type ReactionResponse struct {
	ID        string       `json:"id"`
	Emoji     string       `json:"emoji"`
	User      UserResponse `json:"user"`
	CreatedAt string       `json:"createdAt"`
}

// MessageResponse matches the Message schema
type MessageResponse struct {
	ID                 string             `json:"id"`
	ConversationID     string             `json:"conversationId"`
	Sender             UserResponse       `json:"sender"`
	CreatedAt          string             `json:"createdAt"`
	ContentType        string             `json:"contentType"`
	Text               *string            `json:"text,omitempty"`
	PhotoURL           *string            `json:"photoUrl,omitempty"`
	FileURL            *string            `json:"fileUrl,omitempty"`
	FileName           *string            `json:"fileName,omitempty"`
	RepliedToMessageID *string            `json:"repliedToMessageId,omitempty"`
	Status             string             `json:"status"`
	Reactions          []ReactionResponse `json:"reactions"`
}

// ConversationResponse matches the Conversation schema (full details)
type ConversationResponse struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Title        string            `json:"title"`
	PhotoURL     *string           `json:"photoUrl,omitempty"`
	Participants []UserResponse    `json:"participants"`
	Messages     []MessageResponse `json:"messages"`
}

// GroupResponse matches the Group schema
type GroupResponse struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	PhotoURL *string        `json:"photoUrl,omitempty"`
	Members  []UserResponse `json:"members"`
}

// ============================================================================
// REQUEST TYPES
// ============================================================================

// SendMessageRequest is the request body for POST /conversations/{id}/messages
type SendMessageRequest struct {
	ContentType      string  `json:"contentType"`
	Text             *string `json:"text,omitempty"`
	PhotoURL         *string `json:"photoUrl,omitempty"`
	FileURL          *string `json:"fileUrl,omitempty"`
	FileName         *string `json:"fileName,omitempty"`
	ReplyToMessageID *string `json:"replyToMessageId,omitempty"`
}

// CommentMessageRequest is the request body for POST .../comments (reactions)
type CommentMessageRequest struct {
	Emoji string `json:"emoji"`
}

// ForwardMessageRequest is the request body for POST .../forward
type ForwardMessageRequest struct {
	TargetConversationID string `json:"targetConversationId"`
}

// CreateGroupRequest is the request body for POST /groups
type CreateGroupRequest struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"memberIds,omitempty"`
}

// AddToGroupRequest is the request body for POST /groups/{id}/members
type AddToGroupRequest struct {
	UserID string `json:"userId"`
}

// SetGroupNameRequest is the request body for PUT /groups/{id}/name
type SetGroupNameRequest struct {
	Name string `json:"name"`
}

// ============================================================================
// SESSION / LOGIN ENDPOINTS
// ============================================================================

// doLogin handles POST /session
// - If username exists, return its ID
// - If username doesn't exist, create new user and return its ID
func (rt *_router) doLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	// Validate username length (3-16 characters)
	if len(req.Name) < 3 || len(req.Name) > 16 {
		sendBadRequest(w, "Username must be between 3 and 16 characters")
		return
	}

	// Check if user exists
	user, err := rt.db.GetUserByName(req.Name)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}

	var userID string

	if user != nil {
		// User exists, return existing ID
		userID = user.ID
	} else {
		// User doesn't exist, create new one
		newID, err := uuid.NewV4()
		if err != nil {
			sendInternalError(w, "Error generating ID")
			return
		}
		userID = newID.String()

		if err := rt.db.CreateUser(userID, req.Name); err != nil {
			ctx.Logger.WithError(err).Error("error creating user")
			sendInternalError(w, "Error creating user")
			return
		}
	}

	sendJSON(w, http.StatusCreated, LoginResponse{
		Identifier: userID,
	})
}

// ============================================================================
// CURRENT USER (/me) ENDPOINTS
// ============================================================================

// getMe handles GET /me - returns the current user's profile
func (rt *_router) getMe(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	sendJSON(w, http.StatusOK, UserResponse{
		ID:          user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		PhotoURL:    user.PhotoURL,
	})
}

// setMyUsername handles PUT /me/username - change current user's username
func (rt *_router) setMyUsername(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	var req SetUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	// Validate username length
	if len(req.Name) < 3 || len(req.Name) > 16 {
		sendBadRequest(w, "Username must be between 3 and 16 characters")
		return
	}

	// Check if new username is already taken
	existing, err := rt.db.GetUserByName(req.Name)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if existing != nil && existing.ID != user.ID {
		sendConflict(w, "Username is already taken")
		return
	}

	// Update username
	if err := rt.db.UpdateUsername(user.ID, req.Name); err != nil {
		ctx.Logger.WithError(err).Error("error updating username")
		sendInternalError(w, "Error updating username")
		return
	}

	// Return updated user
	sendJSON(w, http.StatusOK, UserResponse{
		ID:          user.ID,
		Name:        req.Name,
		DisplayName: user.DisplayName,
		PhotoURL:    user.PhotoURL,
	})
}

// ============================================================================
// USERS ENDPOINTS
// ============================================================================

// searchUsers handles GET /users - list or search users
func (rt *_router) searchUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	query := r.URL.Query().Get("q")

	var users []UserResponse

	if query == "" {
		// Return all users
		dbUsers, err := rt.db.GetAllUsers()
		if err != nil {
			ctx.Logger.WithError(err).Error("database error")
			sendInternalError(w, "Database error")
			return
		}
		for _, u := range dbUsers {
			users = append(users, UserResponse{
				ID:          u.ID,
				Name:        u.Name,
				DisplayName: u.DisplayName,
				PhotoURL:    u.PhotoURL,
			})
		}
	} else {
		// Search users by query
		dbUsers, err := rt.db.SearchUsers(query)
		if err != nil {
			ctx.Logger.WithError(err).Error("database error")
			sendInternalError(w, "Database error")
			return
		}
		for _, u := range dbUsers {
			users = append(users, UserResponse{
				ID:          u.ID,
				Name:        u.Name,
				DisplayName: u.DisplayName,
				PhotoURL:    u.PhotoURL,
			})
		}
	}

	// Ensure we return empty array instead of null
	if users == nil {
		users = []UserResponse{}
	}

	sendJSON(w, http.StatusOK, SearchUsersResponse{
		Users: users,
	})
}

// ============================================================================
// CONVERSATION ENDPOINTS
// ============================================================================

// createConversation handles POST /conversations - start a new direct conversation
// Also supports "Message Yourself" feature (like WhatsApp) when userId equals current user's ID
func (rt *_router) createConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	var req CreateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	if req.UserID == "" {
		sendBadRequest(w, "userId is required")
		return
	}

	// Check if this is a self-conversation ("Message Yourself" feature)
	isSelfConversation := req.UserID == user.ID

	// Check if target user exists (for non-self conversations)
	var targetUser *database.User
	var err error
	if isSelfConversation {
		// For self-conversation, use current user as target
		targetUser = user
	} else {
		targetUser, err = rt.db.GetUserByID(req.UserID)
		if err != nil {
			ctx.Logger.WithError(err).Error("database error")
			sendInternalError(w, "Database error")
			return
		}
		if targetUser == nil {
			sendNotFound(w, "User not found")
			return
		}
	}

	// Check if direct conversation already exists
	existingConv, err := rt.db.GetDirectConversation(user.ID, req.UserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}

	if existingConv != nil {
		// Return existing conversation
		participants, _ := rt.db.GetParticipants(existingConv.ID)
		var participantResponses []UserResponse
		for _, p := range participants {
			participantResponses = append(participantResponses, UserResponse{
				ID:          p.ID,
				Name:        p.Name,
				DisplayName: p.DisplayName,
				PhotoURL:    p.PhotoURL,
			})
		}

		// Set title appropriately for self-conversation
		existingTitle := targetUser.Name
		if isSelfConversation {
			existingTitle = "Message Yourself"
		}

		sendJSON(w, http.StatusOK, ConversationResponse{
			ID:           existingConv.ID,
			Type:         existingConv.Type,
			Title:        existingTitle,
			PhotoURL:     targetUser.PhotoURL,
			Participants: participantResponses,
			Messages:     []MessageResponse{},
		})
		return
	}

	// Create new direct conversation
	convID, _ := uuid.NewV4()

	// For self-conversation, set a special name
	convName := ""
	if isSelfConversation {
		convName = "Message Yourself"
	}

	if err := rt.db.CreateConversation(convID.String(), "direct", convName); err != nil {
		ctx.Logger.WithError(err).Error("error creating conversation")
		sendInternalError(w, "Error creating conversation")
		return
	}

	// Add participants (for self-conversation, only add once)
	_ = rt.db.AddParticipant(convID.String(), user.ID)
	if !isSelfConversation {
		_ = rt.db.AddParticipant(convID.String(), req.UserID)
	}

	// Set title for self-conversation
	title := targetUser.Name
	if isSelfConversation {
		title = "Message Yourself"
	}

	// Build participants list
	participants := []UserResponse{
		{ID: user.ID, Name: user.Name, DisplayName: user.DisplayName, PhotoURL: user.PhotoURL},
	}
	if !isSelfConversation {
		participants = append(participants, UserResponse{
			ID: targetUser.ID, Name: targetUser.Name, DisplayName: targetUser.DisplayName, PhotoURL: targetUser.PhotoURL,
		})
	}

	sendJSON(w, http.StatusCreated, ConversationResponse{
		ID:           convID.String(),
		Type:         "direct",
		Title:        title,
		Participants: participants,
		Messages:     []MessageResponse{},
	})
}

// getMyConversations handles GET /conversations - list user's conversations
// Also marks messages from others as "received" (one checkmark)
func (rt *_router) getMyConversations(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	// Update status of messages from others to "received" for all user's conversations
	_ = rt.db.MarkMessagesAsReceived(user.ID)

	summaries, err := rt.db.GetConversationSummariesByUser(user.ID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}

	var response []ConversationSummaryResponse
	for _, s := range summaries {
		// For direct conversations, get the other participant's name as title
		title := s.Title
		if s.Type == "direct" {
			participants, _ := rt.db.GetParticipants(s.ID)
			for _, p := range participants {
				if p.ID != user.ID {
					title = p.Name
					break
				}
			}
		}

		response = append(response, ConversationSummaryResponse{
			ID:                 s.ID,
			Type:               s.Type,
			Title:              title,
			PhotoURL:           s.PhotoURL,
			LastMessageAt:      s.LastMessageAt,
			LastMessageSnippet: s.LastMessageSnippet,
			LastMessageIsPhoto: s.LastMessageIsPhoto,
		})
	}

	if response == nil {
		response = []ConversationSummaryResponse{}
	}

	sendJSON(w, http.StatusOK, response)
}

// getConversation handles GET /conversations/{conversationId} - get conversation with messages
// Also marks messages from others as "read" (two checkmarks)
func (rt *_router) getConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	conversationID := ps.ByName("conversationId")

	// Check if user is participant
	isParticipant, err := rt.db.IsParticipant(conversationID, user.ID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if !isParticipant {
		sendNotFound(w, "Conversation not found or you are not a participant")
		return
	}

	// Mark messages from others as "read" (two checkmarks) since user is viewing the conversation
	_ = rt.db.MarkMessagesAsRead(conversationID, user.ID)

	conv, err := rt.db.GetConversationByID(conversationID)
	if err != nil || conv == nil {
		sendNotFound(w, "Conversation not found")
		return
	}

	// Get participants
	participants, err := rt.db.GetParticipants(conversationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error getting participants")
		sendInternalError(w, "Database error")
		return
	}

	var participantResponses []UserResponse
	for _, p := range participants {
		participantResponses = append(participantResponses, UserResponse{
			ID:          p.ID,
			Name:        p.Name,
			DisplayName: p.DisplayName,
			PhotoURL:    p.PhotoURL,
		})
	}

	// Get messages
	messages, err := rt.db.GetMessagesByConversation(conversationID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error getting messages")
		sendInternalError(w, "Database error")
		return
	}

	var messageResponses []MessageResponse
	for _, m := range messages {
		// Get sender
		sender, _ := rt.db.GetUserByID(m.SenderID)
		var senderResponse UserResponse
		if sender != nil {
			senderResponse = UserResponse{
				ID:          sender.ID,
				Name:        sender.Name,
				DisplayName: sender.DisplayName,
				PhotoURL:    sender.PhotoURL,
			}
		}

		// Get reactions
		reactions, _ := rt.db.GetReactionsByMessage(m.ID)
		var reactionResponses []ReactionResponse
		for _, reaction := range reactions {
			reactUser, _ := rt.db.GetUserByID(reaction.UserID)
			var reactUserResponse UserResponse
			if reactUser != nil {
				reactUserResponse = UserResponse{
					ID:          reactUser.ID,
					Name:        reactUser.Name,
					DisplayName: reactUser.DisplayName,
					PhotoURL:    reactUser.PhotoURL,
				}
			}
			reactionResponses = append(reactionResponses, ReactionResponse{
				ID:        reaction.ID,
				Emoji:     reaction.Emoji,
				User:      reactUserResponse,
				CreatedAt: reaction.CreatedAt,
			})
		}
		if reactionResponses == nil {
			reactionResponses = []ReactionResponse{}
		}

		messageResponses = append(messageResponses, MessageResponse{
			ID:                 m.ID,
			ConversationID:     m.ConversationID,
			Sender:             senderResponse,
			CreatedAt:          m.CreatedAt,
			ContentType:        m.ContentType,
			Text:               m.Text,
			PhotoURL:           m.PhotoURL,
			FileURL:            m.FileURL,
			FileName:           m.FileName,
			RepliedToMessageID: m.RepliedToMessageID,
			Status:             m.Status,
			Reactions:          reactionResponses,
		})
	}

	if messageResponses == nil {
		messageResponses = []MessageResponse{}
	}

	// Determine title
	title := conv.Name
	if conv.Type == "direct" {
		for _, p := range participants {
			if p.ID != user.ID {
				title = p.Name
				break
			}
		}
	}

	sendJSON(w, http.StatusOK, ConversationResponse{
		ID:           conv.ID,
		Type:         conv.Type,
		Title:        title,
		PhotoURL:     conv.PhotoURL,
		Participants: participantResponses,
		Messages:     messageResponses,
	})
}

// sendMessage handles POST /conversations/{conversationId}/messages - send a message
func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	conversationID := ps.ByName("conversationId")

	// Check if user is participant
	isParticipant, err := rt.db.IsParticipant(conversationID, user.ID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if !isParticipant {
		sendNotFound(w, "Conversation not found or you are not a participant")
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	// Validate content type
	validTypes := map[string]bool{"text": true, "photo": true, "audio": true, "document": true, "file": true}
	if !validTypes[req.ContentType] {
		sendBadRequest(w, "contentType must be 'text', 'photo', 'audio', 'document', or 'file'")
		return
	}

	// Validate content
	if req.ContentType == "text" && (req.Text == nil || *req.Text == "") {
		sendBadRequest(w, "text is required for text messages")
		return
	}
	if req.ContentType == "photo" && (req.PhotoURL == nil || *req.PhotoURL == "") {
		sendBadRequest(w, "photoUrl is required for photo messages")
		return
	}
	if (req.ContentType == "audio" || req.ContentType == "document" || req.ContentType == "file") && (req.FileURL == nil || *req.FileURL == "") {
		sendBadRequest(w, "fileUrl is required for audio/document/file messages")
		return
	}

	// Generate message ID and timestamp
	msgID, _ := uuid.NewV4()
	createdAt := globaltime.Now().UTC().Format("2006-01-02T15:04:05Z")

	msg := database.Message{
		ID:                 msgID.String(),
		ConversationID:     conversationID,
		SenderID:           user.ID,
		CreatedAt:          createdAt,
		ContentType:        req.ContentType,
		Text:               req.Text,
		PhotoURL:           req.PhotoURL,
		FileURL:            req.FileURL,
		FileName:           req.FileName,
		RepliedToMessageID: req.ReplyToMessageID,
		Status:             "sent",
	}

	if err := rt.db.CreateMessage(msg); err != nil {
		ctx.Logger.WithError(err).Error("error creating message")
		sendInternalError(w, "Error creating message")
		return
	}

	sendJSON(w, http.StatusCreated, MessageResponse{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		Sender: UserResponse{
			ID:          user.ID,
			Name:        user.Name,
			DisplayName: user.DisplayName,
			PhotoURL:    user.PhotoURL,
		},
		CreatedAt:          msg.CreatedAt,
		ContentType:        msg.ContentType,
		Text:               msg.Text,
		PhotoURL:           msg.PhotoURL,
		FileURL:            msg.FileURL,
		FileName:           msg.FileName,
		RepliedToMessageID: msg.RepliedToMessageID,
		Status:             msg.Status,
		Reactions:          []ReactionResponse{},
	})
}

// ============================================================================
// MESSAGE ENDPOINTS
// ============================================================================

// deleteMessage handles DELETE /conversations/{conversationId}/messages/{messageId}
func (rt *_router) deleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	messageID := ps.ByName("messageId")

	// Get the message
	msg, err := rt.db.GetMessageByID(messageID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if msg == nil {
		sendNotFound(w, "Message not found")
		return
	}

	// Check if user is the sender
	if msg.SenderID != user.ID {
		sendForbidden(w, "You can only delete your own messages")
		return
	}

	if err := rt.db.DeleteMessage(messageID); err != nil {
		ctx.Logger.WithError(err).Error("error deleting message")
		sendInternalError(w, "Error deleting message")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// forwardMessage handles POST /conversations/{conversationId}/messages/{messageId}/forward
func (rt *_router) forwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	messageID := ps.ByName("messageId")

	var req ForwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	// Get original message
	origMsg, err := rt.db.GetMessageByID(messageID)
	if err != nil || origMsg == nil {
		sendNotFound(w, "Original message not found")
		return
	}

	// Check if user is participant of target conversation
	isParticipant, err := rt.db.IsParticipant(req.TargetConversationID, user.ID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if !isParticipant {
		sendNotFound(w, "Target conversation not found or you are not a participant")
		return
	}

	// Create forwarded message
	msgID, _ := uuid.NewV4()
	createdAt := globaltime.Now().UTC().Format("2006-01-02T15:04:05Z")

	newMsg := database.Message{
		ID:             msgID.String(),
		ConversationID: req.TargetConversationID,
		SenderID:       user.ID,
		CreatedAt:      createdAt,
		ContentType:    origMsg.ContentType,
		Text:           origMsg.Text,
		PhotoURL:       origMsg.PhotoURL,
		Status:         "sent",
	}

	if err := rt.db.CreateMessage(newMsg); err != nil {
		ctx.Logger.WithError(err).Error("error creating forwarded message")
		sendInternalError(w, "Error forwarding message")
		return
	}

	sendJSON(w, http.StatusCreated, MessageResponse{
		ID:             newMsg.ID,
		ConversationID: newMsg.ConversationID,
		Sender: UserResponse{
			ID:          user.ID,
			Name:        user.Name,
			DisplayName: user.DisplayName,
			PhotoURL:    user.PhotoURL,
		},
		CreatedAt:   newMsg.CreatedAt,
		ContentType: newMsg.ContentType,
		Text:        newMsg.Text,
		PhotoURL:    newMsg.PhotoURL,
		Status:      newMsg.Status,
		Reactions:   []ReactionResponse{},
	})
}

// commentMessage handles POST /conversations/{conversationId}/messages/{messageId}/comments
func (rt *_router) commentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	messageID := ps.ByName("messageId")

	// Check message exists
	msg, err := rt.db.GetMessageByID(messageID)
	if err != nil || msg == nil {
		sendNotFound(w, "Message not found")
		return
	}

	var req CommentMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	if req.Emoji == "" {
		sendBadRequest(w, "emoji is required")
		return
	}

	reactionID, _ := uuid.NewV4()
	createdAt := globaltime.Now().UTC().Format("2006-01-02T15:04:05Z")

	reaction := database.Reaction{
		ID:        reactionID.String(),
		MessageID: messageID,
		UserID:    user.ID,
		Emoji:     req.Emoji,
		CreatedAt: createdAt,
	}

	if err := rt.db.CreateReaction(reaction); err != nil {
		ctx.Logger.WithError(err).Error("error creating reaction")
		sendInternalError(w, "Error creating reaction")
		return
	}

	sendJSON(w, http.StatusCreated, ReactionResponse{
		ID:    reaction.ID,
		Emoji: reaction.Emoji,
		User: UserResponse{
			ID:          user.ID,
			Name:        user.Name,
			DisplayName: user.DisplayName,
			PhotoURL:    user.PhotoURL,
		},
		CreatedAt: reaction.CreatedAt,
	})
}

// uncommentMessage handles DELETE /conversations/{conversationId}/messages/{messageId}/comments/{commentId}
func (rt *_router) uncommentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	commentID := ps.ByName("commentId")

	// Get the reaction
	reaction, err := rt.db.GetReactionByID(commentID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if reaction == nil {
		sendNotFound(w, "Reaction not found")
		return
	}

	// Check if user is the author
	if reaction.UserID != user.ID {
		sendForbidden(w, "You can only delete your own reactions")
		return
	}

	if err := rt.db.DeleteReaction(commentID); err != nil {
		ctx.Logger.WithError(err).Error("error deleting reaction")
		sendInternalError(w, "Error deleting reaction")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// GROUP ENDPOINTS
// ============================================================================

// createGroup handles POST /groups - create a new group
func (rt *_router) createGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	if req.Name == "" {
		sendBadRequest(w, "name is required")
		return
	}

	groupID, _ := uuid.NewV4()

	// Create the group conversation
	if err := rt.db.CreateConversation(groupID.String(), "group", req.Name); err != nil {
		ctx.Logger.WithError(err).Error("error creating group")
		sendInternalError(w, "Error creating group")
		return
	}

	// Add creator as participant
	if err := rt.db.AddParticipant(groupID.String(), user.ID); err != nil {
		ctx.Logger.WithError(err).Error("error adding creator to group")
		sendInternalError(w, "Error creating group")
		return
	}

	// Add initial members
	for _, memberID := range req.MemberIDs {
		_ = rt.db.AddParticipant(groupID.String(), memberID)
	}

	// Get all members for response
	members, _ := rt.db.GetParticipants(groupID.String())
	var memberResponses []UserResponse
	for _, m := range members {
		memberResponses = append(memberResponses, UserResponse{
			ID:          m.ID,
			Name:        m.Name,
			DisplayName: m.DisplayName,
			PhotoURL:    m.PhotoURL,
		})
	}

	sendJSON(w, http.StatusCreated, GroupResponse{
		ID:      groupID.String(),
		Name:    req.Name,
		Members: memberResponses,
	})
}

// getGroup handles GET /groups/{groupId}
func (rt *_router) getGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	groupID := ps.ByName("groupId")

	// Check if user is member
	isMember, err := rt.db.IsParticipant(groupID, user.ID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if !isMember {
		sendNotFound(w, "Group not found or you are not a member")
		return
	}

	conv, err := rt.db.GetConversationByID(groupID)
	if err != nil || conv == nil || conv.Type != "group" {
		sendNotFound(w, "Group not found")
		return
	}

	members, _ := rt.db.GetParticipants(groupID)
	var memberResponses []UserResponse
	for _, m := range members {
		memberResponses = append(memberResponses, UserResponse{
			ID:          m.ID,
			Name:        m.Name,
			DisplayName: m.DisplayName,
			PhotoURL:    m.PhotoURL,
		})
	}

	sendJSON(w, http.StatusOK, GroupResponse{
		ID:       conv.ID,
		Name:     conv.Name,
		PhotoURL: conv.PhotoURL,
		Members:  memberResponses,
	})
}

// addToGroup handles POST /groups/{groupId}/members
func (rt *_router) addToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	groupID := ps.ByName("groupId")

	// Check if requester is member
	isMember, err := rt.db.IsParticipant(groupID, user.ID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if !isMember {
		sendNotFound(w, "Group not found or you are not a member")
		return
	}

	var req AddToGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	// Check if user to add exists
	userToAdd, err := rt.db.GetUserByID(req.UserID)
	if err != nil || userToAdd == nil {
		sendNotFound(w, "User to add not found")
		return
	}

	if err := rt.db.AddParticipant(groupID, req.UserID); err != nil {
		ctx.Logger.WithError(err).Error("error adding user to group")
		sendInternalError(w, "Error adding user to group")
		return
	}

	// Return updated group
	conv, _ := rt.db.GetConversationByID(groupID)
	members, _ := rt.db.GetParticipants(groupID)
	var memberResponses []UserResponse
	for _, m := range members {
		memberResponses = append(memberResponses, UserResponse{
			ID:          m.ID,
			Name:        m.Name,
			DisplayName: m.DisplayName,
			PhotoURL:    m.PhotoURL,
		})
	}

	sendJSON(w, http.StatusOK, GroupResponse{
		ID:       conv.ID,
		Name:     conv.Name,
		PhotoURL: conv.PhotoURL,
		Members:  memberResponses,
	})
}

// leaveGroup handles DELETE /groups/{groupId}/members/me
func (rt *_router) leaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	groupID := ps.ByName("groupId")

	// Check if user is member
	isMember, err := rt.db.IsParticipant(groupID, user.ID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if !isMember {
		sendNotFound(w, "Group not found or you are not a member")
		return
	}

	if err := rt.db.RemoveParticipant(groupID, user.ID); err != nil {
		ctx.Logger.WithError(err).Error("error leaving group")
		sendInternalError(w, "Error leaving group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// setGroupName handles PUT /groups/{groupId}/name
func (rt *_router) setGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	groupID := ps.ByName("groupId")

	// Check if user is member
	isMember, err := rt.db.IsParticipant(groupID, user.ID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if !isMember {
		sendNotFound(w, "Group not found or you are not a member")
		return
	}

	var req SetGroupNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendBadRequest(w, "Invalid JSON")
		return
	}

	if err := rt.db.UpdateConversationName(groupID, req.Name); err != nil {
		ctx.Logger.WithError(err).Error("error updating group name")
		sendInternalError(w, "Error updating group name")
		return
	}

	conv, _ := rt.db.GetConversationByID(groupID)
	members, _ := rt.db.GetParticipants(groupID)
	var memberResponses []UserResponse
	for _, m := range members {
		memberResponses = append(memberResponses, UserResponse{
			ID:          m.ID,
			Name:        m.Name,
			DisplayName: m.DisplayName,
			PhotoURL:    m.PhotoURL,
		})
	}

	sendJSON(w, http.StatusOK, GroupResponse{
		ID:       conv.ID,
		Name:     conv.Name,
		PhotoURL: conv.PhotoURL,
		Members:  memberResponses,
	})
}

// ============================================================================
// PHOTO UPLOAD ENDPOINTS
// ============================================================================

// setMyPhoto handles PUT /me/photo - upload profile photo
func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		sendBadRequest(w, "Invalid multipart form or file too large")
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		sendBadRequest(w, "photo file is required")
		return
	}
	defer file.Close()

	// In a real app, you'd save the file to storage (S3, local disk, etc.)
	// For now, we'll just store a placeholder URL
	photoURL := "/uploads/users/" + user.ID + "/" + header.Filename

	if err := rt.db.UpdateUserPhoto(user.ID, &photoURL); err != nil {
		ctx.Logger.WithError(err).Error("error updating user photo")
		sendInternalError(w, "Error updating photo")
		return
	}

	sendJSON(w, http.StatusOK, UserResponse{
		ID:          user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		PhotoURL:    &photoURL,
	})
}

// setGroupPhoto handles PUT /groups/{groupId}/photo
func (rt *_router) setGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		sendUnauthorized(w, "User not found in context")
		return
	}

	groupID := ps.ByName("groupId")

	// Check if user is member
	isMember, err := rt.db.IsParticipant(groupID, user.ID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		sendInternalError(w, "Database error")
		return
	}
	if !isMember {
		sendNotFound(w, "Group not found or you are not a member")
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		sendBadRequest(w, "Invalid multipart form or file too large")
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		sendBadRequest(w, "photo file is required")
		return
	}
	defer file.Close()

	// Placeholder URL
	photoURL := "/uploads/groups/" + groupID + "/" + header.Filename

	if err := rt.db.UpdateConversationPhoto(groupID, &photoURL); err != nil {
		ctx.Logger.WithError(err).Error("error updating group photo")
		sendInternalError(w, "Error updating photo")
		return
	}

	conv, _ := rt.db.GetConversationByID(groupID)
	members, _ := rt.db.GetParticipants(groupID)
	var memberResponses []UserResponse
	for _, m := range members {
		memberResponses = append(memberResponses, UserResponse{
			ID:          m.ID,
			Name:        m.Name,
			DisplayName: m.DisplayName,
			PhotoURL:    m.PhotoURL,
		})
	}

	sendJSON(w, http.StatusOK, GroupResponse{
		ID:       conv.ID,
		Name:     conv.Name,
		PhotoURL: &photoURL,
		Members:  memberResponses,
	})
}
