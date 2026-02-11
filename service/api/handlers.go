package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

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
	IsForwarded        bool               `json:"isForwarded"`
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
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	PhotoURL  *string        `json:"photoUrl,omitempty"`
	CreatedBy string         `json:"createdBy"`
	Members   []UserResponse `json:"members"`
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

// setMyUserName handles PUT /me/username - change current user's username
func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
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

	// Broadcast profile update to all users who have conversations with this user
	userConversations, err := rt.db.GetConversationSummariesByUser(user.ID)
	if err == nil {
		uniqueParticipants := make(map[string]bool)
		for _, conv := range userConversations {
			participants, _ := rt.db.GetParticipants(conv.ID)
			for _, p := range participants {
				if p.ID != user.ID {
					uniqueParticipants[p.ID] = true
				}
			}
		}
		var participantIDs []string
		for id := range uniqueParticipants {
			participantIDs = append(participantIDs, id)
		}
		rt.wsHub.BroadcastToUsers(participantIDs, WebSocketMessage{
			Type: "profile_updated",
			Payload: map[string]interface{}{
				"userId":   user.ID,
				"name":     req.Name,
				"photoUrl": user.PhotoURL,
			},
		})
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

// startConversation handles POST /conversations - start a new direct conversation
// Also supports "Message Yourself" feature (like WhatsApp) when userId equals current user's ID
func (rt *_router) startConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
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

	if err := rt.db.CreateConversation(convID.String(), "direct", convName, nil); err != nil {
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

	conversationResponse := ConversationResponse{
		ID:           convID.String(),
		Type:         "direct",
		Title:        title,
		Participants: participants,
		Messages:     []MessageResponse{},
	}

	// Broadcast new conversation to both participants (for real-time conversations list update)
	rt.wsHub.SendToUser(user.ID, WebSocketMessage{
		Type:    "new_conversation",
		Payload: conversationResponse,
	})
	if !isSelfConversation {
		rt.wsHub.SendToUser(req.UserID, WebSocketMessage{
			Type:    "new_conversation",
			Payload: conversationResponse,
		})
	}

	sendJSON(w, http.StatusCreated, conversationResponse)
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
		// For direct conversations, get the other participant's name and photo as title/photo
		title := s.Title
		photoURL := s.PhotoURL
		if s.Type == "direct" {
			participants, _ := rt.db.GetParticipants(s.ID)
			for _, p := range participants {
				if p.ID != user.ID {
					title = p.Name
					photoURL = p.PhotoURL
					break
				}
			}
		}

		response = append(response, ConversationSummaryResponse{
			ID:                 s.ID,
			Type:               s.Type,
			Title:              title,
			PhotoURL:           photoURL,
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

	// Notify message senders that their messages have been read
	// For groups, only notify when ALL members have read the message
	participantsForRead, _ := rt.db.GetParticipants(conversationID)
	if len(participantsForRead) > 0 {
		// Get all messages in this conversation
		messages, err := rt.db.GetMessagesByConversation(conversationID)
		if err == nil {
			// Check which messages are now fully read by everyone
			var fullyReadMessageIDs []string
			for _, msg := range messages {
				status, err := rt.db.GetMessageStatus(msg.ID)
				if err == nil && status == database.StatusRead {
					fullyReadMessageIDs = append(fullyReadMessageIDs, msg.ID)
				}
			}

			// Broadcast to all other participants with the updated message statuses
			var senderIDs []string
			for _, p := range participantsForRead {
				if p.ID != user.ID {
					senderIDs = append(senderIDs, p.ID)
				}
			}
			rt.wsHub.BroadcastToUsers(senderIDs, WebSocketMessage{
				Type: "messages_read",
				Payload: map[string]interface{}{
					"conversationId":      conversationID,
					"readByUserId":        user.ID,
					"fullyReadMessageIds": fullyReadMessageIDs,
				},
			})
		}
	}

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

	// Optimize: Collect all unique user IDs from messages
	userIDSet := make(map[string]bool)
	for _, p := range participants {
		userIDSet[p.ID] = true
	}
	for _, m := range messages {
		userIDSet[m.SenderID] = true
	}

	// Optimize: Fetch all reactions for this conversation at once
	allReactions, err := rt.db.GetReactionsByConversation(conversationID)
	if err != nil {
		ctx.Logger.WithError(err).Warn("error fetching reactions")
		allReactions = []database.Reaction{}
	}

	// Group reactions by message ID
	reactionsByMessage := make(map[string][]database.Reaction)
	for _, r := range allReactions {
		reactionsByMessage[r.MessageID] = append(reactionsByMessage[r.MessageID], r)
		userIDSet[r.UserID] = true
	}

	// Optimize: Fetch all users at once
	var userIDs []string
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}
	users, err := rt.db.GetUsersByIDs(userIDs)
	if err != nil {
		ctx.Logger.WithError(err).Warn("error fetching users in batch")
		users = []database.User{}
	}

	// Create user map for O(1) lookups
	userMap := make(map[string]database.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	var messageResponses []MessageResponse
	for _, m := range messages {
		// Get sender from map
		sender := userMap[m.SenderID]
		senderResponse := UserResponse{
			ID:          sender.ID,
			Name:        sender.Name,
			DisplayName: sender.DisplayName,
			PhotoURL:    sender.PhotoURL,
		}

		// Get reactions from map
		reactions := reactionsByMessage[m.ID]
		var reactionResponses []ReactionResponse
		for _, reaction := range reactions {
			reactUser := userMap[reaction.UserID]
			reactUserResponse := UserResponse{
				ID:          reactUser.ID,
				Name:        reactUser.Name,
				DisplayName: reactUser.DisplayName,
				PhotoURL:    reactUser.PhotoURL,
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

		// Calculate dynamic status based on read receipts
		messageStatus, err := rt.db.GetMessageStatus(m.ID)
		if err != nil {
			ctx.Logger.WithError(err).Warn("error calculating message status, using stored status")
			messageStatus = m.Status
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
			Status:             messageStatus,
			Reactions:          reactionResponses,
			IsForwarded:        m.IsForwarded,
		})
	}

	if messageResponses == nil {
		messageResponses = []MessageResponse{}
	}

	// Determine title and photoURL for direct conversations
	title := conv.Name
	photoURL := conv.PhotoURL
	if conv.Type == "direct" {
		for _, p := range participants {
			if p.ID != user.ID {
				title = p.Name
				photoURL = p.PhotoURL
				break
			}
		}
	}

	sendJSON(w, http.StatusOK, ConversationResponse{
		ID:           conv.ID,
		Type:         conv.Type,
		Title:        title,
		PhotoURL:     photoURL,
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

	// Validate content - text and photo can co-exist
	if req.ContentType == "text" && (req.Text == nil || *req.Text == "") && (req.PhotoURL == nil || *req.PhotoURL == "") {
		sendBadRequest(w, "text or photoUrl is required for text messages")
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
		Status:             database.StatusSent,
		IsForwarded:        false,
	}

	if err := rt.db.CreateMessage(msg); err != nil {
		ctx.Logger.WithError(err).Error("error creating message")
		sendInternalError(w, "Error creating message")
		return
	}

	messageResponse := MessageResponse{
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
		IsForwarded:        msg.IsForwarded,
	}

	// Broadcast new message to all conversation participants via WebSocket
	participants, err := rt.db.GetParticipants(conversationID)
	if err == nil && len(participants) > 0 {
		var participantIDs []string
		for _, p := range participants {
			participantIDs = append(participantIDs, p.ID)
		}
		// Broadcast new message for ChatView
		rt.wsHub.BroadcastToUsers(participantIDs, WebSocketMessage{
			Type:    "new_message",
			Payload: messageResponse,
		})
		// Broadcast conversation update for ConversationsView (so list updates with new snippet)
		rt.wsHub.BroadcastToUsers(participantIDs, WebSocketMessage{
			Type: "conversation_updated",
			Payload: map[string]interface{}{
				"conversationId":     conversationID,
				"lastMessageSnippet": msg.Text,
				"lastMessageIsPhoto": msg.ContentType == "photo",
				"lastMessageAt":      msg.CreatedAt,
			},
		})
	}

	sendJSON(w, http.StatusCreated, messageResponse)
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
		FileURL:        origMsg.FileURL,
		FileName:       origMsg.FileName,
		Status:         database.StatusSent,
		IsForwarded:    true,
	}

	if err := rt.db.CreateMessage(newMsg); err != nil {
		ctx.Logger.WithError(err).Error("error creating forwarded message")
		sendInternalError(w, "Error forwarding message")
		return
	}

	messageResponse := MessageResponse{
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
		FileURL:     newMsg.FileURL,
		FileName:    newMsg.FileName,
		Status:      newMsg.Status,
		Reactions:   []ReactionResponse{},
		IsForwarded: newMsg.IsForwarded,
	}

	// Broadcast forwarded message to all participants in target conversation via WebSocket
	participants, err := rt.db.GetParticipants(req.TargetConversationID)
	if err == nil && len(participants) > 0 {
		var participantIDs []string
		for _, p := range participants {
			participantIDs = append(participantIDs, p.ID)
		}
		// Broadcast new message for ChatView
		rt.wsHub.BroadcastToUsers(participantIDs, WebSocketMessage{
			Type:    "new_message",
			Payload: messageResponse,
		})
		// Broadcast conversation update for ConversationsView
		rt.wsHub.BroadcastToUsers(participantIDs, WebSocketMessage{
			Type: "conversation_updated",
			Payload: map[string]interface{}{
				"conversationId":     req.TargetConversationID,
				"lastMessageSnippet": newMsg.Text,
				"lastMessageIsPhoto": newMsg.ContentType == "photo",
				"lastMessageAt":      newMsg.CreatedAt,
			},
		})
	}

	sendJSON(w, http.StatusCreated, messageResponse)
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

	reactionResponse := ReactionResponse{
		ID:    reaction.ID,
		Emoji: reaction.Emoji,
		User: UserResponse{
			ID:          user.ID,
			Name:        user.Name,
			DisplayName: user.DisplayName,
			PhotoURL:    user.PhotoURL,
		},
		CreatedAt: reaction.CreatedAt,
	}

	// Broadcast reaction to all conversation participants via WebSocket
	conversationID := ps.ByName("conversationId")
	participants, err := rt.db.GetParticipants(conversationID)
	if err == nil {
		var participantIDs []string
		for _, p := range participants {
			participantIDs = append(participantIDs, p.ID)
		}
		rt.wsHub.BroadcastToUsers(participantIDs, WebSocketMessage{
			Type: "reaction_added",
			Payload: map[string]interface{}{
				"conversationId": conversationID,
				"messageId":      messageID,
				"reaction":       reactionResponse,
			},
		})
	}

	sendJSON(w, http.StatusCreated, reactionResponse)
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

	// Broadcast reaction removal to all conversation participants via WebSocket
	conversationID := ps.ByName("conversationId")
	participants, err := rt.db.GetParticipants(conversationID)
	if err == nil {
		var participantIDs []string
		for _, p := range participants {
			participantIDs = append(participantIDs, p.ID)
		}
		rt.wsHub.BroadcastToUsers(participantIDs, WebSocketMessage{
			Type: "reaction_removed",
			Payload: map[string]interface{}{
				"conversationId": conversationID,
				"messageId":      reaction.MessageID,
				"reactionId":     commentID,
				"userId":         user.ID,
			},
		})
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
	if err := rt.db.CreateConversation(groupID.String(), "group", req.Name, &user.ID); err != nil {
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

	groupResponse := GroupResponse{
		ID:        groupID.String(),
		Name:      req.Name,
		CreatedBy: user.ID,
		Members:   memberResponses,
	}

	// Broadcast new group to all participants (creator + members) for real-time conversations list update
	allParticipantIDs := append([]string{user.ID}, req.MemberIDs...)
	for _, participantID := range allParticipantIDs {
		rt.wsHub.SendToUser(participantID, WebSocketMessage{
			Type: "new_conversation",
			Payload: ConversationResponse{
				ID:           groupID.String(),
				Type:         "group",
				Title:        req.Name,
				PhotoURL:     nil,
				Participants: memberResponses,
				Messages:     []MessageResponse{},
			},
		})
	}

	sendJSON(w, http.StatusCreated, groupResponse)
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

	// Get createdBy value, handling nil case
	var createdBy string
	if conv.CreatedBy != nil {
		createdBy = *conv.CreatedBy
	}

	sendJSON(w, http.StatusOK, GroupResponse{
		ID:        conv.ID,
		Name:      conv.Name,
		PhotoURL:  conv.PhotoURL,
		CreatedBy: createdBy,
		Members:   memberResponses,
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

	// Get createdBy value, handling nil case
	var createdBy string
	if conv.CreatedBy != nil {
		createdBy = *conv.CreatedBy
	}

	sendJSON(w, http.StatusOK, GroupResponse{
		ID:        conv.ID,
		Name:      conv.Name,
		PhotoURL:  conv.PhotoURL,
		CreatedBy: createdBy,
		Members:   memberResponses,
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

	// Get createdBy value, handling nil case
	var createdBy string
	if conv.CreatedBy != nil {
		createdBy = *conv.CreatedBy
	}

	// Broadcast group update to all members via WebSocket
	var memberIDs []string
	for _, m := range members {
		memberIDs = append(memberIDs, m.ID)
	}
	rt.wsHub.BroadcastToUsers(memberIDs, WebSocketMessage{
		Type: "group_updated",
		Payload: map[string]interface{}{
			"groupId":  groupID,
			"name":     conv.Name,
			"photoUrl": conv.PhotoURL,
		},
	})

	sendJSON(w, http.StatusOK, GroupResponse{
		ID:        conv.ID,
		Name:      conv.Name,
		PhotoURL:  conv.PhotoURL,
		CreatedBy: createdBy,
		Members:   memberResponses,
	})
}

// ============================================================================
// PHOTO UPLOAD ENDPOINTS
// ============================================================================

// uploadMessagePhoto handles POST /conversations/{conversationId}/photos - upload photo for message
func (rt *_router) uploadMessagePhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
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

	// Save file to uploads directory
	uploadDir := "./uploads/messages/" + conversationID
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		ctx.Logger.WithError(err).Error("error creating upload directory")
		sendInternalError(w, "Error saving photo")
		return
	}

	// Generate unique filename to avoid conflicts
	fileID, _ := uuid.NewV4()
	fileName := fileID.String() + "_" + header.Filename
	filePath := uploadDir + "/" + fileName

	dst, err := os.Create(filePath)
	if err != nil {
		ctx.Logger.WithError(err).Error("error creating file")
		sendInternalError(w, "Error saving photo")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		ctx.Logger.WithError(err).Error("error saving file")
		sendInternalError(w, "Error saving photo")
		return
	}

	photoURL := "/uploads/messages/" + conversationID + "/" + fileName

	sendJSON(w, http.StatusOK, map[string]string{
		"photoUrl": photoURL,
	})
}

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

	// Save file to uploads directory
	uploadDir := "./uploads/users/" + user.ID
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		ctx.Logger.WithError(err).Error("error creating upload directory")
		sendInternalError(w, "Error saving photo")
		return
	}

	filePath := uploadDir + "/" + header.Filename
	dst, err := os.Create(filePath)
	if err != nil {
		ctx.Logger.WithError(err).Error("error creating file")
		sendInternalError(w, "Error saving photo")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		ctx.Logger.WithError(err).Error("error saving file")
		sendInternalError(w, "Error saving photo")
		return
	}

	photoURL := "/uploads/users/" + user.ID + "/" + header.Filename

	if err := rt.db.UpdateUserPhoto(user.ID, &photoURL); err != nil {
		ctx.Logger.WithError(err).Error("error updating user photo")
		sendInternalError(w, "Error updating photo")
		return
	}

	// Broadcast profile photo update to all users who have conversations with this user
	userConversations, err := rt.db.GetConversationSummariesByUser(user.ID)
	if err == nil && len(userConversations) > 0 {
		// Collect all unique participant IDs from all conversations
		uniqueParticipants := make(map[string]bool)
		for _, conv := range userConversations {
			participants, _ := rt.db.GetParticipants(conv.ID)
			for _, p := range participants {
				if p.ID != user.ID {
					uniqueParticipants[p.ID] = true
				}
			}
		}
		// Broadcast to all participants
		var participantIDs []string
		for id := range uniqueParticipants {
			participantIDs = append(participantIDs, id)
		}
		rt.wsHub.BroadcastToUsers(participantIDs, WebSocketMessage{
			Type: "profile_updated",
			Payload: map[string]interface{}{
				"userId":   user.ID,
				"name":     user.Name,
				"photoUrl": photoURL,
			},
		})
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

	// Save file to uploads directory
	uploadDir := "./uploads/groups/" + groupID
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		ctx.Logger.WithError(err).Error("error creating upload directory")
		sendInternalError(w, "Error saving photo")
		return
	}

	filePath := uploadDir + "/" + header.Filename
	dst, err := os.Create(filePath)
	if err != nil {
		ctx.Logger.WithError(err).Error("error creating file")
		sendInternalError(w, "Error saving photo")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		ctx.Logger.WithError(err).Error("error saving file")
		sendInternalError(w, "Error saving photo")
		return
	}

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

	// Get createdBy value, handling nil case
	var createdBy string
	if conv.CreatedBy != nil {
		createdBy = *conv.CreatedBy
	}

	// Broadcast group photo update to all members via WebSocket
	var memberIDs []string
	for _, m := range members {
		memberIDs = append(memberIDs, m.ID)
	}
	rt.wsHub.BroadcastToUsers(memberIDs, WebSocketMessage{
		Type: "group_updated",
		Payload: map[string]interface{}{
			"groupId":  groupID,
			"name":     conv.Name,
			"photoUrl": photoURL,
		},
	})

	sendJSON(w, http.StatusOK, GroupResponse{
		ID:        conv.ID,
		Name:      conv.Name,
		PhotoURL:  &photoURL,
		CreatedBy: createdBy,
		Members:   memberResponses,
	})
}
